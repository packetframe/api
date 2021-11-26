package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/common/db"
	"github.com/packetframe/api/internal/common/util"
	"github.com/packetframe/api/internal/orchestrator/metrics"
)

// TODO: Sentry

// Linker flags
var version = "dev"

const (
	startupDelay   = 5 * time.Second
	updateInterval = 2 * time.Second
)

var (
	database *gorm.DB
	edges    map[string]string
	queue    []string
)

type config struct {
	RPCListen      string `env:"RPC_LISTEN,required"`
	MetricsListen  string `env:"METRICS_LISTEN,required"`
	DbHost         string `env:"DB_HOST,required"`
	CacheDirectory string `env:"CACHE_DIR,required"`
	SSHKeyFile     string `env:"SSH_KEY_FILE,required"`
	SSHPort        uint32 `env:"SSH_PORT,required"`
	NodeFile       string `env:"NODE_FILE,required"`
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

// purgeZoneFiles removes all zone files on disk that aren't referenced in the database
func purgeZoneFiles() error {
	zones, err := db.ZoneList(database)
	if err != nil {
		return err
	}
	log.Debugf("Found %d zones", len(zones))

	// Remove files that aren't referenced in the database
	files, err := os.ReadDir(conf.CacheDirectory)
	if err != nil {
		return err
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
			if err := os.Remove(path.Join(conf.CacheDirectory, f.Name())); err != nil {
				log.Warnf("removing %s: %s", f.Name(), err)
			}
		}
	}

	return nil
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
	zoneFile := fmt.Sprintf(`@ IN SOA ns1.packetframe.com. info.packetframe.com. %d 7200 3600 1209600 300
@ 86400 IN NS ns1.packetframe.com.
@ 86400 IN NS ns2.packetframe.com.
`, zone.Serial)

	for _, record := range records {
		zoneFile += fmt.Sprintf("%s %d IN %s %s\n", record.Label, record.TTL, record.Type, record.Value)
	}

	// Write the zone file to disk
	return os.WriteFile(path.Join(conf.CacheDirectory, "db."+strings.TrimSuffix(zone.Zone, ".")), []byte(zoneFile), 0644)
}

// deployZoneFile copies a zone file to all edge nodes and returns a boolean if all edge nodes received the transfer correctly
func deployZoneFile(zoneId string) (bool, error) {
	zone, err := db.ZoneFindByID(database, zoneId)
	if err != nil {
		return false, err
	}

	transferOk := true
	for host, ip := range edges {
		log.Infof("Attempting deploy zone to %s (%s)", host, ip)
		cmd := exec.Command("rsync",
			"--delete",
			"--progress",
			"--partial",
			"--archive",
			"--compress",
			"-e", fmt.Sprintf("ssh -J vpn.fmt2 -p %d -i %s", conf.SSHPort, conf.SSHKeyFile),
			path.Join(conf.CacheDirectory, "db."+strings.TrimSuffix(zone.Zone, ".")),
			"root@"+ip+":/opt/packetframe/dns/zones/")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		log.Debugf("Running %s", cmd.String())
		if err := cmd.Run(); err != nil {
			transferOk = false
			log.Warnf("zone deploy to %s (%s): %v", host, ip, err)
		}
	}
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

	log.Println("Connecting to database")
	database, err = gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=api password=api dbname=api port=5432 sslmode=disable", conf.DbHost)), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/update_zone", func(w http.ResponseWriter, r *http.Request) {
		elem := "zone_update/" + r.URL.Query().Get("id")
		if !util.StrSliceContains(queue, elem) {
			queue = append(queue, elem)
		}
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
				if strings.HasPrefix(message, "zone_update") {
					zoneId := strings.Split(message, "/")[1]
					log.Debugf("Updating zone %s", zoneId)

					transferOk := true
					if err := buildZoneFile(zoneId); err != nil {
						transferOk = false
						log.Warn(err)
					}
					ok, err := deployZoneFile(zoneId)
					if err != nil {
						log.Warn(err)
					}
					if !ok || err != nil {
						transferOk = false
					}

					// TODO: This might break the for loop
					if transferOk {
						queue = queue[1:]
					}
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
