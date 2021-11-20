package orchestrator

import (
	"time"

	log "github.com/sirupsen/logrus"
)

const updateInterval = 30 * time.Second

var update = false

// Register creates a new goroutine to update nodes every updateInterval
func Register() {
	suffixUpdateTicker := time.NewTicker(updateInterval)
	go func() {
		for range suffixUpdateTicker.C {
			if update {
				update = false
				log.Debugln("Sending node update")
				run()
			}
		}
	}()
}

// Update queues a new update
func Update() {
	update = true
}

// run runs a new full system node update
func run() {
	// TODO
}
