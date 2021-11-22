package metrics

import (
	"gorm.io/gorm"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/packetframe/api/internal/common/db"
)

var (
	metricUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "packetframe_api_users",
		Help: "Total user accounts",
	})
	metricZones = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "packetframe_api_zones",
		Help: "Total DNS zones",
	})
)

// Collector runs a ticker to periodically update metrics values
func Collector(database *gorm.DB, interval time.Duration) {
	metricsTicker := time.NewTicker(interval)
	for range metricsTicker.C {
		users, err := db.UserList(database)
		if err != nil {
			log.Warn(err)
		}
		metricUsers.Set(float64(len(users)))

		zones, err := db.ZoneList(database)
		if err != nil {
			log.Warn(err)
		}
		metricZones.Set(float64(len(zones)))
	}
}

// Listen starts the HTTP metrics listener
func Listen(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	log.Infof("Starting metrics exporter on http://%s/metrics", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
