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
)

func Serve(address string) {
	log.Infof("Prometheus serving metrics at %v", address)
	_ = http.ListenAndServe(address, promhttp.Handler())
}

func InitializeMetrics() {
	PrivilegedUsersPerGateway = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "naisdevice",
		Subsystem: "apiserver",
		Name:      "privileged_users",
		Help:      "privileged users per gateway",
	}, []string{"gateway"})
	prometheus.MustRegister(PrivilegedUsersPerGateway)

	DeviceConfigsReturned = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "naisdevice",
		Subsystem:   "apiserver",
		Name:        "user_configs_returned",
		Help:        "Total number of configs returned to device since apiserver started.",
	}, []string{"serial", "username"})
	prometheus.MustRegister(DeviceConfigsReturned)

	GatewayConfigsReturned = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "naisdevice",
		Subsystem: "apiserver",
		Name:      "gateway_configs_returned",
		Help:      "Total number of configs returned to gateway since apiserver started.",
	}, []string{"gateway"})
	prometheus.MustRegister(GatewayConfigsReturned)
}
