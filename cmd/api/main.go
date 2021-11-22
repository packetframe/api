package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	log "github.com/sirupsen/logrus"

	"github.com/packetframe/api/internal/api/routes"
	"github.com/packetframe/api/internal/api/validation"
	"github.com/packetframe/api/internal/common/db"
)

// Linker flags
var version = "dev"

func main() {
	dbHost := os.Getenv("DB_HOST")
	log.Infof("DB host %s", dbHost)
	postgresDSN := fmt.Sprintf("host=%s user=api password=api dbname=api port=5432 sslmode=disable", os.Getenv("DB_HOST"))

	if version == "dev" {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Running in dev mode")
	}

	if os.Getenv("DOCUMENT") != "" {
		fmt.Println(routes.Document())
		os.Exit(0)
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
	suffixUpdateTicker := time.NewTicker(24 * time.Hour)
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
	routes.Register(app)

	if err := validation.Register(); err != nil {
		log.Fatal(err)
	}

	listenAddr := ":8080"
	log.Printf("Starting API on %s", listenAddr)
	log.Fatal(app.Listen(listenAddr))
}
