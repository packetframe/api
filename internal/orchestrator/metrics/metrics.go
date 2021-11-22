package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	MetricLastUpdated = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "packetframe_orchestrator_last_update",
		Help: "Timestamp of the end of the last orchestrator update run",
	})
)

// Listen starts the HTTP metrics listener
func Listen(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	log.Infof("Starting metrics exporter on http://%s/metrics", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
