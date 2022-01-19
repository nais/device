package apiserver_metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	DevicesConnected          prometheus.Gauge
	PrivilegedUsersPerGateway *prometheus.GaugeVec
	DeviceConfigsReturned     *prometheus.CounterVec
	GatewayConfigsReturned    *prometheus.CounterVec
)

func Serve(address string) error {
	return http.ListenAndServe(address, promhttp.Handler())
}

func init() {
	DevicesConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "naisdevice",
		Subsystem: "apiserver",
		Name:      "devices_connected",
		Help:      "number of clients currently connected to api server",
	})

	PrivilegedUsersPerGateway = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "naisdevice",
		Subsystem: "apiserver",
		Name:      "privileged_users",
		Help:      "privileged users per gateway",
	}, []string{"gateway"})

	DeviceConfigsReturned = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "naisdevice",
		Subsystem: "apiserver",
		Name:      "user_configs_returned",
		Help:      "Total number of configs returned to device since apiserver started.",
	}, []string{"serial", "username"})

	GatewayConfigsReturned = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "naisdevice",
		Subsystem: "apiserver",
		Name:      "gateway_configs_returned",
		Help:      "Total number of configs returned to gateway since apiserver started.",
	}, []string{"gateway"})

	prometheus.MustRegister(
		DevicesConnected,
		PrivilegedUsersPerGateway,
		DeviceConfigsReturned,
		GatewayConfigsReturned,
	)
}
