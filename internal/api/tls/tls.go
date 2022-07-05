package tls

import (
	"context"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/caddyserver/certmagic"
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"

	"github.com/packetframe/api/internal/common/db"
)

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

	return nil, nil
}

var (
	certMagicConfig *certmagic.Config
	issuer          *certmagic.ACMEIssuer
)

// Init initializes the ACME validation server
func Init(dataDir string) {
	certMagicConfig = certmagic.NewDefault()
	certMagicConfig.Storage = &certmagic.FileStorage{Path: dataDir}
	issuer = certmagic.NewACMEIssuer(certMagicConfig, certmagic.ACMEIssuer{
		CA:     certmagic.LetsEncryptStagingCA,
		Email:  "tls@packetframe.com",
		Agreed: true,
	})
	certMagicConfig.Issuers = []certmagic.Issuer{issuer}
}

// Serve starts the ACME validation server
func Serve(host string, database *gorm.DB) {
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
			if err := certMagicConfig.ManageSync(context.TODO(), domains); err != nil {
				sentry.CaptureException(err)
				log.Warnf("Failed to sync certificates: %v", err)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	log.Infof("Starting ACME validation server on %s", host)
	log.Fatal(http.ListenAndServe(host, issuer.HTTPChallengeHandler(mux)))
}
