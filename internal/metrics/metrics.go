package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog"
	"net/http"
)

var (
	filesUploaded = promauto.NewCounter(prometheus.CounterOpts{
		Name: "aqua_files_uploaded_total",
		Help: "The total number of files uploaded",
	})

	filesExpired = promauto.NewCounter(prometheus.CounterOpts{
		Name: "aqua_files_expired_total",
		Help: "The total number of files expired",
	})
)

// StartMetricsServer starts the internal Prometheus metrics server
// which enables us to seperate traffic from this endpoint and the normal file
// serving.
func StartMetricsServer() {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	addr := ":8766"

	klog.Infof("Starting the metrics server on %s", addr)
	err := http.ListenAndServe(addr, mux)
	if err != nil {
		klog.Warningf("Could not start the metrics server: %v", err)
		return
	}
}

func IncFilesUploaded() {
	filesUploaded.Inc()
}

func IncFilesExpired() {
	filesExpired.Inc()
}
