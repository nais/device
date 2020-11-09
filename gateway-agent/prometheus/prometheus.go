package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

var (
	FailedConfigFetches       prometheus.Counter
	LastSuccessfulConfigFetch prometheus.Gauge
	RegisteredDevices         prometheus.Gauge
	ConnectedDevices          prometheus.Gauge
	CurrentVersion            prometheus.Counter
)

func Serve(address string) {
	go func() {
		log.Infof("Prometheus serving metrics at %v", address)
		_ = http.ListenAndServe(address, promhttp.Handler())
	}()
}

func InitializeMetrics(name, version string) {
	CurrentVersion = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "current_version",
		Help:        "current running version",
		Namespace:   "naisdevice",
		Subsystem:   "gateway_agent",
		ConstLabels: prometheus.Labels{"name": name, "version": version},
	})
	FailedConfigFetches = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "failed_config_fetches",
		Help:        "count of failed config fetches",
		Namespace:   "naisdevice",
		Subsystem:   "gateway_agent",
		ConstLabels: prometheus.Labels{"name": name, "version": version},
	})
	LastSuccessfulConfigFetch = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "last_successful_config_fetch",
		Help:        "time since last successful config fetch",
		Namespace:   "naisdevice",
		Subsystem:   "gateway_agent",
		ConstLabels: prometheus.Labels{"name": name, "version": version},
	})
	RegisteredDevices = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "number_of_registered_devices",
		Help:        "number of registered devices on a gateway",
		Namespace:   "naisdevice",
		Subsystem:   "gateway_agent",
		ConstLabels: prometheus.Labels{"name": name, "version": version},
	})
	ConnectedDevices = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "number_of_connected_devices",
		Help:        "number of connected devices on a gateway",
		Namespace:   "naisdevice",
		Subsystem:   "gateway_agent",
		ConstLabels: prometheus.Labels{"name": name, "version": version},
	})

	prometheus.MustRegister(FailedConfigFetches)
	prometheus.MustRegister(LastSuccessfulConfigFetch)
	prometheus.MustRegister(RegisteredDevices)
	prometheus.MustRegister(ConnectedDevices)
	prometheus.MustRegister(CurrentVersion)
}
