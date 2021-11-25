package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/common/db"
	"github.com/packetframe/api/internal/orchestrator/metrics"
)

// Linker flags
var version = "dev"

const updateInterval = 30 * time.Second

var (
	database       *gorm.DB
	dbHost         = os.Getenv("DB_HOST")
	cacheDirectory = os.Getenv("CACHE_DIR")
	metricsListen  = os.Getenv("METRICS_LISTEN")
)

func update() {
	zones, err := db.ZoneList(database)
	if err != nil {
		log.Fatal(err)
	}
	log.Debugf("Found %d zones", len(zones))

	for _, zone := range zones {
		zone, err := db.ZoneFindByID(database, zone.ID)
		if err != nil {
			log.Fatal(err)
		}

		records, err := db.RecordList(database, zone.ID)
		if err != nil {
			log.Fatal(err)
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
		if err := os.WriteFile(path.Join(cacheDirectory, "db."+strings.TrimSuffix(zone.Zone, ".")), []byte(zoneFile), 0644); err != nil {
			log.Fatal(err)
		}
	}

	// Remove files that aren't referenced in the database
	files, err := ioutil.ReadDir(cacheDirectory)
	if err != nil {
		log.Fatal(err)
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
			if err := os.Remove(path.Join(cacheDirectory, f.Name())); err != nil {
				log.Warnf("removing %s: %s", f.Name(), err)
			}
		}
	}

	metrics.MetricLastUpdated.Set(float64(time.Now().Unix()))
}

func main() {
	log.Infof("Starting Packetframe zone orchestrator (%s)", version)

	if dbHost == "" {
		log.Fatal("DB_HOST must be set")
	}
	if cacheDirectory == "" {
		log.Fatal("CACHE_DIR must be set")
	}
	if metricsListen == "" {
		log.Fatal("METRICS_LISTEN must be set")
	}

	// Make cache directory
	if err := os.MkdirAll(cacheDirectory, os.ModeDir); err != nil {
		log.Fatal(err)
	}

	log.Infof("DB host %s, cache %s", dbHost, cacheDirectory)
	postgresDSN := fmt.Sprintf("host=%s user=api password=api dbname=api port=5432 sslmode=disable", os.Getenv("DB_HOST"))

	if version == "dev" {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Running in dev mode")
	} else {
		startupDelay := 5 * time.Second
		log.Printf("Waiting %+v before connecting to database...", startupDelay)
		time.Sleep(startupDelay)
	}

	log.Println("Connecting to database")
	var err error
	database, err = gorm.Open(postgres.Open(postgresDSN), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Metrics listener
	go metrics.Listen(metricsListen)

	if version == "dev" {
		log.Info("Dev mode enabled, updating once and exiting")
		update()
	} else {
		// Update local zone cache
		log.Infof("Starting update ticker every %+v", updateInterval)
		zoneFileUpdateTicker := time.NewTicker(updateInterval)
		for range zoneFileUpdateTicker.C {
			update()
		}
	}
}
