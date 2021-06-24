package api

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

var (
	PrivilegedUsersPerGateway *prometheus.GaugeVec
	DeviceConfigsReturned     *prometheus.CounterVec
	GatewayConfigsReturned    *prometheus.CounterVec
	initialized = false
)

func Serve(address string) {
	log.Infof("Prometheus serving metrics at %v", address)
	_ = http.ListenAndServe(address, promhttp.Handler())
}

func InitializeMetrics() {
	if initialized {
		return
	}
	initialized = true

	PrivilegedUsersPerGateway = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "naisdevice",
		Subsystem: "apiserver",
		Name:      "privileged_users",
		Help:      "privileged users per gateway",
	}, []string{"gateway"})

	DeviceConfigsReturned = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "naisdevice",
		Subsystem:   "apiserver",
		Name:        "user_configs_returned",
		Help:        "Total number of configs returned to device since apiserver started.",
	}, []string{"serial", "username"})

	GatewayConfigsReturned = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "naisdevice",
		Subsystem: "apiserver",
		Name:      "gateway_configs_returned",
		Help:      "Total number of configs returned to gateway since apiserver started.",
	}, []string{"gateway"})

	prometheus.MustRegister(PrivilegedUsersPerGateway, DeviceConfigsReturned, GatewayConfigsReturned)
}
