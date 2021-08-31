package main

import (
	"flag"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/routes"
)

var (
	listenAddr  = flag.String("l", ":8080", "API listen address")
	postgresDSN = flag.String("p", "host=localhost user=api password=api dbname=api port=5432 sslmode=disable", "Postgres DSN")
)

func main() {
	flag.Parse()

	log.Println("Connecting to database")
	database, err := db.Connect(*postgresDSN)
	if err != nil {
		log.Fatal(err)
	}
	routes.Database = database

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	routes.Register(app)

	log.Printf("Starting API on %s", *listenAddr)
	log.Fatal(app.Listen(*listenAddr))
}