package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/common/db"
)

var version = "dev"

// proxiedDomains is a list of domains that are configured to be proxied
func proxiedDomains(database *gorm.DB) ([]string, error) {
	var domains []string

	zones, err := db.ZoneList(database)
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		records, err := db.RecordList(database, zone.ID)
		if err != nil {
			return nil, err
		}

		for _, record := range records {
			if record.Proxy {
				domain := record.Label
				if domain == "@" {
					domain = zone.Zone
				} else if !strings.HasSuffix(domain, zone.Zone) {
					domain += "." + zone.Zone
				}
				domain = strings.TrimSuffix(domain, ".")
				domains = append(domains, domain)
			}
		}
	}

	return domains, nil
}

// TODO: Delete credentials that are no longer needed.
// Steps:
// Find credentials on disk that don't have any proxied records and delete them
// Delete credential rows from database that don't have a file on disk

func main() {
	dbHost := os.Getenv("DB_HOST")
	sentryDsn := os.Getenv("SENTRY_DSN")
	listen := os.Getenv("LISTEN")
	dataDir := os.Getenv("DATA_DIR")
	verbose := os.Getenv("VERBOSE")
	ca := os.Getenv("CA")

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
	if ca == "" {
		ca = certmagic.LetsEncryptStagingCA
		log.Warnf("CA is not set, defaulting to %s", ca)
	}

	log.Println("Connecting to database")
	database, err := db.Open(fmt.Sprintf("host=%s user=api password=api dbname=api port=5432 sslmode=disable", dbHost))
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

	certMagicConfig := certmagic.NewDefault()
	certMagicConfig.Storage = &certmagic.FileStorage{Path: dataDir}
	issuer := certmagic.NewACMEIssuer(certMagicConfig, certmagic.ACMEIssuer{
		CA:     ca,
		Email:  "tls@packetframe.com",
		Agreed: true,
	})
	certMagicConfig.Issuers = []certmagic.Issuer{issuer}

	if certMagicConfig == nil || issuer == nil {
		msg := "ACME server not initialized"
		sentry.CaptureMessage(msg)
		log.Fatal(msg)
	}

	syncTicker := time.NewTicker(24 * time.Hour)
	go func() {
		for ; true; <-syncTicker.C {
			log.Debug("Attempting certificate sync")
			domains, err := proxiedDomains(database)
			if err != nil {
				sentry.CaptureException(err)
				log.Warnf("Failed to get proxied domains: %v", err)
			}
			log.Debugf("Found proxied domains: %v", domains)
			if err := certMagicConfig.ManageSync(context.TODO(), domains); err != nil {
				sentry.CaptureException(err)
				log.Warnf("Failed to sync certificates: %v", err)
			}
			log.Debug("Certificate sync complete")

			log.Debug("Adding certificates to database")
			certDir := path.Join(dataDir, "certificates", strings.ReplaceAll(strings.TrimPrefix(ca, "https://"), "/", "-"))
			log.Debugf("Looking for certificates in %s", certDir)
			dirs, err := filepath.Glob(certDir + "/*")
			if err != nil {
				sentry.CaptureException(err)
				log.Warnf("Failed to get certificate files: %v", err)
			}
			for _, d := range dirs {
				domain := strings.TrimPrefix(d, certDir+"/")

				// Load the keypair
				certFile, err := os.ReadFile(path.Join(certDir, domain, domain+".crt"))
				if err != nil {
					sentry.CaptureException(err)
					log.Warnf("Failed to read certificate file for %s: %v", domain, err)
				}
				keyFile, err := os.ReadFile(path.Join(certDir, domain, domain+".key"))
				if err != nil {
					sentry.CaptureException(err)
					log.Warnf("Failed to read key file for %s: %v", domain, err)
				}

				// Skip empty domains
				if domain == "" || string(certFile) == "" || string(keyFile) == "" {
					continue
				}

				if err := db.CredentialAddOrUpdate(database, domain, string(certFile), string(keyFile)); err != nil {
					sentry.CaptureException(err)
					log.Warnf("Failed to add certificate for %s: %v", domain, err)
				}
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	log.Infof("Starting ACME validation server on %s", listen)
	log.Fatal(http.ListenAndServe(listen, issuer.HTTPChallengeHandler(mux)))
}
