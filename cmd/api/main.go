package main

import (
	"fmt"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	log "github.com/sirupsen/logrus"

	"github.com/packetframe/api/internal/api/metrics"
	"github.com/packetframe/api/internal/api/routes"
	"github.com/packetframe/api/internal/api/validation"
	"github.com/packetframe/api/internal/common/db"
)

// Linker flags
var (
	version = "dev"
	commit  = "dev"
	date    = "dev"
)

const (
	suffixListUpdateInterval = 24 * time.Hour
	metricsUpdateInterval    = 15 * time.Minute
)

var (
	dbHost        = os.Getenv("DB_HOST")
	metricsListen = os.Getenv("METRICS_LISTEN")

	smtpHost = os.Getenv("SMTP_HOST")
	smtpUser = os.Getenv("SMTP_USER")
	smtpPass = os.Getenv("SMTP_PASS")

	sentryDsn = os.Getenv("SENTRY_DSN")
)

func main() {
	if os.Getenv("DOCUMENT") != "" {
		fmt.Println(routes.Document())
		os.Exit(0)
	}

	log.Infof("Starting Packetframe API (%s)", version)

	if dbHost == "" {
		log.Fatal("DB_HOST must be set")
	}
	if metricsListen == "" {
		log.Fatal("METRICS_LISTEN must be set")
	}
	if smtpHost == "" {
		log.Fatal("SMTP_HOST must be set")
	}
	if smtpUser == "" {
		log.Fatal("SMTP_USER must be set")
	}
	if smtpPass == "" {
		log.Fatal("SMTP_PASS must be set")
	}
	if sentryDsn == "" {
		log.Fatalf("SENTRY_DSN must be set")
	}

	routes.SMTPHost = smtpHost
	routes.SMTPUser = smtpUser
	routes.SMTPPass = smtpPass

	log.Infof("DB host %s", dbHost)
	postgresDSN := fmt.Sprintf("host=%s user=api password=api dbname=api port=5432 sslmode=disable", dbHost)

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:     sentryDsn,
		Release: version,
	}); err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	if version == "dev" {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Running in dev mode")
	} else {
		startupDelay := 5 * time.Second
		log.Printf("Waiting %+v before connecting to database...", startupDelay)
		time.Sleep(startupDelay)
	}

	log.Println("Connecting to database")
	database, err := db.Connect(postgresDSN)
	if err != nil {
		log.Fatal(err)
	}
	routes.Database = database

	// Update public suffix list once
	log.Debugln("Updating local public suffix list")
	routes.Suffixes, err = db.SuffixList()
	if err != nil {
		log.Fatal(err)
	}

	// Update public suffix list on a ticker
	suffixUpdateTicker := time.NewTicker(suffixListUpdateInterval)
	go func() {
		for range suffixUpdateTicker.C {
			log.Debugln("Updating local public suffix list")
			routes.Suffixes, err = db.SuffixList()
			if err != nil {
				log.Fatal(err)
			}
		}
	}()

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	if version == "dev" {
		log.Debugln("Adding wildcard CORS origin")
		app.Use(cors.New(cors.Config{
			AllowOrigins:     "http://localhost:3000",
			AllowCredentials: true,
		}))
	}
	routes.Register(app, map[string]interface{}{"version": version, "commit": commit, "date": date})

	if err := validation.Register(); err != nil {
		log.Fatal(err)
	}

	// Metrics goroutines
	go metrics.Collector(database, metricsUpdateInterval)
	go metrics.Listen(metricsListen)

	startupMessage := fmt.Sprintf("Starting Packetframe API v%s (%s) on :8080", version, commit)
	sentry.CaptureMessage(startupMessage)
	log.Println(startupMessage)
	log.Fatal(app.Listen(":8080"))
}
