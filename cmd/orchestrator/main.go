package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/common/db"
	"github.com/packetframe/api/internal/orchestrator/metrics"
)

// TODO: Sentry

// Linker flags
var version = "dev"

const (
	startupDelay    = 5 * time.Second
	updateInterval  = 2 * time.Second
	messageLifespan = 30 * time.Minute // Duration after which a queue message will be discarded
	sshOpts         = "-o StrictHostKeyChecking=no"
)

const (
	opZoneUpdate = iota
	opManifestUpdate
	opZonePurge
)

type queueMessage struct {
	operation int
	arg       string
	acked     bool
	created   time.Time
}

var (
	database *gorm.DB
	edges    map[string]string
	queue    []*queueMessage
)

type config struct {
	RPCListen      string `env:"RPC_LISTEN,required"`
	MetricsListen  string `env:"METRICS_LISTEN,required"`
	DbHost         string `env:"DB_HOST,required"`
	CacheDirectory string `env:"CACHE_DIR,required"`
	SSHKeyFile     string `env:"SSH_KEY_FILE,required"`
	SSHPort        uint32 `env:"SSH_PORT,required"`
	NodeFile       string `env:"NODE_FILE,required"`
	NS1Hostname    string `env:"NS1_HOSTNAME"`
	NS2Hostname    string `env:"NS2_HOSTNAME"`
}

var conf config

// parseEdgeConfig parses an edge config file to return a map of node label to IP address
func parseEdgeConfig() (map[string]string, error) {
	var config struct {
		Nodes map[string]string `yaml:"nodes"`
	}

	nodeFileBytes, err := os.ReadFile(conf.NodeFile)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(nodeFileBytes, &config); err != nil {
		return nil, err
	}

	return config.Nodes, nil
}

// purgeZoneFiles removes all zone files on disk that aren't referenced in the database and copies the entire zones directory to all nodes
func purgeZoneFiles() (bool, error) {
	zones, err := db.ZoneList(database)
	if err != nil {
		return false, err
	}
	log.Debugf("Found %d zones", len(zones))

	// Remove files that aren't referenced in the database
	files, err := os.ReadDir(path.Join(conf.CacheDirectory, "zones"))
	if err != nil {
		return false, err
	}
	for _, f := range files {
		found := false
		for _, zone := range zones {
			if "db."+strings.TrimSuffix(zone.Zone, ".") == f.Name() {
				found = true
				break
			}
		}

		if !found {
			log.Debugf("%s not found, removing", f.Name())
			if err := os.Remove(path.Join(conf.CacheDirectory, "zones", f.Name())); err != nil {
				log.Warnf("removing %s: %s", f.Name(), err)
			}
		}
	}

	transferOk := true
	for host, ip := range edges {
		log.Infof("Attempting deploy all zones to %s (%s)", host, ip)
		cmd := exec.Command("rsync",
			"--delete",
			"--progress",
			"--partial",
			"--archive",
			"--compress",
			"-e", fmt.Sprintf("ssh %s -p %d -i %s", sshOpts, conf.SSHPort, conf.SSHKeyFile),
			path.Join(conf.CacheDirectory, "zones/"),
			"root@"+ip+":/opt/packetframe/dns/")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Debugf("Running %s", cmd.String())
		if err := cmd.Run(); err != nil {
			transferOk = false
			log.Warnf("all zones deploy to %s (%s): %v", host, ip, err)
		}

		if ok := reloadKnot(host, ip); !ok {
			transferOk = false
		}
	}

	log.Infoln("Finished purging zone files")
	return transferOk, nil
}

// buildZoneFile writes a zone file to disk by zone ID
func buildZoneFile(zoneID string) error {
	zone, err := db.ZoneFindByID(database, zoneID)
	if err != nil {
		return err
	}

	records, err := db.RecordList(database, zoneID)
	if err != nil {
		return err
	}

	// Serial
	// Refresh, number of seconds after which secondary NSes should query the main to detect zone changes
	// Retry, number of seconds after which secondary NSes should retry serial query from the main if it doesn't respond
	// Expire, number of seconds after which secondary NSes should stop answering if main doesn't respond
	// Negative Cache TTL
	zoneFile := fmt.Sprintf(`@ IN SOA %s.packetframe.com. info.packetframe.com. %d 7200 3600 1209600 300
@ 86400 IN NS %s.packetframe.com.
@ 86400 IN NS %s.packetframe.com.
`, conf.NS1Hostname, zone.Serial, conf.NS1Hostname, conf.NS2Hostname)

	for _, record := range records {
		zoneFile += fmt.Sprintf("%s %d IN %s %s\n", record.Label, record.TTL, record.Type, record.Value)
	}

	// Write the zone file to disk
	return os.WriteFile(path.Join(conf.CacheDirectory, "zones/db."+strings.TrimSuffix(zone.Zone, ".")), []byte(zoneFile), 0644)
}

// reloadKnot reloads the knot daemon
func reloadKnot(host, ip string) bool {
	log.Infof("Reloading knot on %s (%s)", host, ip)
	cmd := exec.Command("ssh",
		sshOpts,
		"-p", strconv.Itoa(int(conf.SSHPort)),
		"-i", conf.SSHKeyFile,
		"root@"+ip,
		"systemctl reload knot")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Debugf("Running %s", cmd.String())
	if err := cmd.Run(); err != nil {
		log.Warnf("Knot reload %s (%s): %v", host, ip, err)
		return false
	}
	return true
}

// deployZoneFile copies a zone file to all edge nodes and returns if all edge nodes received the transfer correctly
func deployZoneFile(zoneId string) (bool, error) {
	zone, err := db.ZoneFindByID(database, zoneId)
	if err != nil {
		return false, err
	}

	transferOk := true
	for host, ip := range edges {
		log.Infof("Attempting deploy %s to %s (%s)", zone.Zone, host, ip)
		cmd := exec.Command("rsync",
			"--delete",
			"--progress",
			"--partial",
			"--archive",
			"--compress",
			"-e", fmt.Sprintf("ssh %s -p %d -i %s", sshOpts, conf.SSHPort, conf.SSHKeyFile),
			path.Join(conf.CacheDirectory, "zones/db."+strings.TrimSuffix(zone.Zone, ".")),
			"root@"+ip+":/opt/packetframe/dns/zones/")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Debugf("Running %s", cmd.String())
		if err := cmd.Run(); err != nil {
			transferOk = false
			log.Warnf("zone deploy to %s (%s): %v", host, ip, err)
		}

		if ok := reloadKnot(host, ip); !ok {
			transferOk = false
		}
	}
	log.Infof("Finished deploying zone file for %s", zone.Zone)

	return transferOk, nil
}

// buildDeployManifest builds and deploys a new Manifest and returns if all edge nodes received the transfer correctly
func buildDeployManifest() (bool, error) {
	zones, err := db.ZoneList(database)
	if err != nil {
		return false, err
	}

	manifestContent := fmt.Sprintf("# knot.zones.conf generated at %v\n", time.Now().UTC())
	for _, zone := range zones {
		manifestContent += fmt.Sprintf(`zone:
  - domain: %s
    template: default
`, strings.TrimSuffix(zone.Zone, "."))
	}

	// Write the zone manifest to disk
	if err := os.WriteFile(path.Join(conf.CacheDirectory, "knot.zones.conf"), []byte(manifestContent), 0644); err != nil {
		return false, err
	}

	transferOk := true
	for host, ip := range edges {
		log.Infof("Attempting deploy knot.zones.conf to %s (%s)", host, ip)
		cmd := exec.Command("rsync",
			"--delete",
			"--progress",
			"--partial",
			"--archive",
			"--compress",
			"-e", fmt.Sprintf("ssh %s -p %d -i %s", sshOpts, conf.SSHPort, conf.SSHKeyFile),
			path.Join(conf.CacheDirectory, "knot.zones.conf"),
			"root@"+ip+":/opt/packetframe/dns/knot.zones.conf")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Debugf("Running %s", cmd.String())
		if err := cmd.Run(); err != nil {
			transferOk = false
			log.Warnf("knot.zones.conf deploy to %s (%s): %v", host, ip, err)
		}

		if ok := reloadKnot(host, ip); !ok {
			transferOk = false
		}
	}
	log.Infoln("Finished deploying knot.zones.conf")

	return transferOk, nil
}

func main() {
	log.Infof("Starting Packetframe zone orchestrator (%s)", version)

	// Parse config from env
	if err := env.Parse(&conf); err != nil {
		log.Fatal(err)
	}
	log.Infof("Config: %+v", conf)

	if //goland:noinspection ALL
	version == "dev" {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Running in dev mode")
	} else {
		log.Printf("Waiting %+v before connecting to database...", startupDelay)
		time.Sleep(startupDelay)
	}

	// Parse edge nodes file
	var err error
	edges, err = parseEdgeConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Loaded %d edge nodes", len(edges))

	// Start watcher for node file
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					edges, err = parseEdgeConfig()
					if err != nil {
						log.Fatal(err)
					}
					log.Infof("Loaded %d edge nodes", len(edges))
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
	err = watcher.Add(conf.NodeFile)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connecting to database")
	database, err = gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=api password=api dbname=api port=5432 sslmode=disable", conf.DbHost)), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/update_zone", func(w http.ResponseWriter, r *http.Request) {
		zoneId := r.URL.Query().Get("id")

		duplicateMessageExists := false
		for _, message := range queue {
			if message.operation == opZoneUpdate && message.arg == zoneId && !message.acked {
				duplicateMessageExists = true
			}
		}

		if !duplicateMessageExists {
			queue = append(queue, &queueMessage{
				operation: opZoneUpdate,
				arg:       zoneId,
				acked:     false,
				created:   time.Now(),
			})
		}
	})

	http.HandleFunc("/update_manifest", func(w http.ResponseWriter, r *http.Request) {
		duplicateMessageExists := false
		for _, message := range queue {
			if message.operation == opManifestUpdate && !message.acked {
				duplicateMessageExists = true
			}
		}

		if !duplicateMessageExists {
			log.Debug("Adding manifest update message")
			queue = append(queue, &queueMessage{
				operation: opManifestUpdate,
				acked:     false,
				created:   time.Now(),
			})
		}
	})

	http.HandleFunc("/purge_zones", func(w http.ResponseWriter, r *http.Request) {
		duplicateMessageExists := false
		for _, message := range queue {
			if message.operation == opZonePurge && !message.acked {
				duplicateMessageExists = true
			}
		}

		if !duplicateMessageExists {
			log.Debug("Adding zone purge message")
			queue = append(queue, &queueMessage{
				operation: opZonePurge,
				acked:     false,
				created:   time.Now(),
			})
		}
	})

	http.HandleFunc("/clear_queue", func(w http.ResponseWriter, r *http.Request) {
		queue = []*queueMessage{}
		fmt.Fprint(w, "Queue cleared")
	})

	http.HandleFunc("/queue_content", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Queue content: %+v", queue)
	})

	// Metrics listener
	go metrics.Listen(conf.MetricsListen)

	// Start update ticker
	log.Infof("Starting update ticker with interval %+v", updateInterval)
	go func() {
		zoneFileUpdateTicker := time.NewTicker(updateInterval)
		for range zoneFileUpdateTicker.C {
			log.Debug("Iterating over queue")
			for _, message := range queue {
				message.acked = true

				// Remove messages created more than messageLifespan ago
				if message.created.After(time.Now().Add(messageLifespan)) {
					log.Debugf("Message created after %s, skipping. opcode %d, arg %s", messageLifespan, message.operation, message.arg)
					queue = queue[1:]
					continue
				}

				switch message.operation {
				case opZoneUpdate:
					if message.arg == "" {
						log.Warn("Got zone update with empty zone arg, skipping")
						// TODO: This might break the for loop
						queue = queue[1:]
						continue
					}

					log.Infof("Updating zone %s", message.arg)

					transferOk := true
					if err := buildZoneFile(message.arg); err != nil {
						transferOk = false
						log.Warn(err)
					}
					ok, err := deployZoneFile(message.arg)
					if err != nil {
						log.Warn(err)
					}
					if !ok || err != nil {
						transferOk = false
					}

					// TODO: This might break the for loop
					if transferOk {
						queue = queue[1:]
					} else {
						// Release the message to be retried
						message.acked = false
					}
				case opManifestUpdate:
					log.Infof("Updating manifest")
					ok, err := buildDeployManifest()
					if err != nil {
						log.Warn(err)
					}

					// TODO: This might break the for loop
					if ok && err == nil {
						queue = queue[1:]
					}
				case opZonePurge:
					log.Infof("Purging zones")
					ok, err := purgeZoneFiles()
					if err != nil {
						log.Warn(err)
					}

					// TODO: This might break the for loop
					if ok && err == nil {
						queue = queue[1:]
					}
				default:
					log.Warnf("Queue message opcode %d not found", message.operation)
				}
			}
			metrics.MetricQueueLength.Set(float64(len(queue)))
		}
	}()

	// Start RPC server
	log.Infof("Starting RPC server on %s", conf.RPCListen)
	if err := http.ListenAndServe(conf.RPCListen, nil); err != nil {
		log.Fatal(err)
	}
}
