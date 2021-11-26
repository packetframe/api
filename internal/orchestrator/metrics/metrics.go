package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var (
	MetricQueueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "packetframe_orchestrator_queue_length",
		Help: "Number of elements in the orchestrator operation queue",
	})
)

// Listen starts the HTTP metrics listener
func Listen(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	log.Infof("Starting metrics exporter on http://%s/metrics", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
