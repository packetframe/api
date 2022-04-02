package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/edged/scriptdns"
)

var (
	dnsListenAddr   = flag.String("dns-listen", ":5354", "DNS listen address")
	rpcListenAddr   = flag.String("rpc-listen", ":8083", "RPC listen address")
	dbHost          = flag.String("db-host", "localhost", "postgres database host")
	refreshInterval = flag.String("refresh", "30s", "script refresh interval")
	verbose         = flag.Bool("verbose", false, "enable verbose logging")
)

func main() {
	flag.Parse()
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	log.Println("Connecting to database")
	database, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=api password=api dbname=api port=5432 sslmode=disable", *dbHost)), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Update public suffix list on a ticker
	refresh, err := time.ParseDuration(*refreshInterval)
	if err != nil {
		log.Fatal(err)
	}
	refreshTicker := time.NewTicker(refresh)
	go func() {
		for range refreshTicker.C {
			log.Debug("Refreshing")
			scriptdns.LoadRecordHandlers(database)
		}
	}()

	log.Printf("Starting ScriptDNS server on %s", *dnsListenAddr)
	scriptdns.Listen(*dnsListenAddr)

	http.HandleFunc("/scriptdns/refresh", func(w http.ResponseWriter, r *http.Request) {
		scriptdns.LoadRecordHandlers(database)
		fmt.Fprint(w, "Refreshed")
	})

	log.Infof("Starting RPC server on %s", *rpcListenAddr)
	if err := http.ListenAndServe(*rpcListenAddr, nil); err != nil {
		log.Fatal(err)
	}
}
