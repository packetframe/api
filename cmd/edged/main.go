package main

import (
	"flag"
	"fmt"
	"gorm.io/gorm"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"

	"github.com/packetframe/api/internal/edged/caddy"
	"github.com/packetframe/api/internal/edged/scriptdns"
	"github.com/packetframe/api/internal/edged/zonegen"
)

var (
	nodeId                = flag.String("node-id", "DEV", "Node ID")
	dnsListenAddr         = flag.String("dns-listen", ":5354", "DNS listen address")
	rpcListenAddr         = flag.String("rpc-listen", ":8083", "RPC listen address")
	dbHost                = flag.String("db-host", "localhost", "Postgres database host")
	zonesDirectory        = flag.String("zones-dir", "/opt/packetframe/dns/zones/", "Directory to store DNS zone files to")
	knotZonesFile         = flag.String("knot-zones-file", "/opt/packetframe/dns/knot.zones.conf", "File to write DNS zone manifest to")
	caddyFile             = flag.String("caddyfile", "", "Path to Caddyfile, disables Caddy functionality if empty")
	certDir               = flag.String("cert-dir", "/opt/packetframe/certs/", "TLS certificate directory")
	scriptRefreshInterval = flag.String("script-refresh", "5s", "Script refresh interval")
	zoneRefreshInterval   = flag.String("zone-refresh", "5s", "Zone refresh interval")
	caddyRefreshInterval  = flag.String("caddy-refresh", "5s", "Caddy refresh interval")
	verbose               = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	log.Println("Connecting to database")
	database, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=readonly password=readonly dbname=api port=5432 sslmode=disable", *dbHost)), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Update public suffix list on a ticker
	scriptRefresh, err := time.ParseDuration(*scriptRefreshInterval)
	if err != nil {
		log.Fatal(err)
	}
	scriptRefreshTicker := time.NewTicker(scriptRefresh)
	go func() {
		for range scriptRefreshTicker.C {
			log.Debug("Refreshing SCRIPT handlers")
			scriptdns.LoadRecordHandlers(database)
		}
	}()

	// Update zones list on a ticker
	zoneRefresh, err := time.ParseDuration(*zoneRefreshInterval)
	if err != nil {
		log.Fatal(err)
	}
	zoneRefreshTicker := time.NewTicker(zoneRefresh)
	go func() {
		for range zoneRefreshTicker.C {
			log.Debug("Refreshing zones")
			if err := zonegen.Update(*zonesDirectory, *knotZonesFile, database); err != nil {
				log.Warnf("zonegen update: %s", err)
			}
		}
	}()

	if *caddyFile != "" {
		log.Info("Caddy enabled")
		caddyRefresh, err := time.ParseDuration(*caddyRefreshInterval)
		if err != nil {
			log.Fatal(err)
		}
		caddyRefreshTicker := time.NewTicker(caddyRefresh)
		go func() {
			for range caddyRefreshTicker.C {
				log.Debug("Refreshing Caddy")
				if err := caddy.Update(database, *caddyFile, *nodeId, *certDir); err != nil {
					log.Warnf("caddy update: %s", err)
				}
			}
		}()
	} else {
		log.Info("Caddy disabled")
	}

	log.Printf("Starting SCRIPT DNS server on %s", *dnsListenAddr)
	scriptdns.Listen(*dnsListenAddr)

	log.Infof("Starting RPC server on %s", *rpcListenAddr)
	if err := http.ListenAndServe(*rpcListenAddr, nil); err != nil {
		log.Fatal(err)
	}
}
