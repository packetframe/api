package main

import (
	"fmt"
	"os"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/api/tls"
)

var version = "dev"

func main() {
	dbHost := os.Getenv("DB_HOST")
	sentryDsn := os.Getenv("SENTRY_DSN")
	listen := os.Getenv("LISTEN")
	dataDir := os.Getenv("DATA_DIR")
	verbose := os.Getenv("VERBOSE")

	if verbose != "" {
		log.SetLevel(log.DebugLevel)
	}

	if dbHost == "" {
		log.Fatal("DB_HOST is not set")
	}
	if sentryDsn == "" {
		log.Fatal("SENTRY_DSN is not set")
	}
	if listen == "" {
		log.Fatal("LISTEN is not set")
	}
	if dataDir == "" {
		log.Fatal("DATA_DIR is not set")
	}

	log.Println("Connecting to database")
	database, err := gorm.Open(postgres.Open(fmt.Sprintf("host=%s user=readonly password=readonly dbname=api port=5432 sslmode=disable", dbHost)), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	if version != "dev" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:     sentryDsn,
			Release: version,
		}); err != nil {
			log.Fatalf("sentry.Init: %s", err)
		}
	} else {
		log.Warn("Version is dev, not starting sentry")
	}

	// ACME validation server
	tls.Init(dataDir)
	tls.Serve(listen, database)
}
