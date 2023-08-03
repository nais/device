package apiserver_metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	DeviceConfigsReturned     *prometheus.CounterVec
	DevicesConnected          prometheus.Gauge
	GatewayConfigsReturned    *prometheus.CounterVec
	PrivilegedUsersPerGateway *prometheus.GaugeVec

	gatewayStatus *prometheus.GaugeVec
)

func Serve(address string) error {
	return http.ListenAndServe(address, promhttp.Handler())
}

func SetConnectedGateways(allGateways, connectedGateways []string) {
	for _, gateway := range allGateways {
		value := 0.0
		for _, conntectedGateway := range connectedGateways {
			if gateway == conntectedGateway {
				value = 1.0
				break
			}
		}
		gatewayStatus.With(prometheus.Labels{
			"gateway": gateway,
		}).Set(value)
	}
}

func init() {
	DevicesConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "naisdevice",
		Subsystem: "apiserver",
		Name:      "devices_connected",
		Help:      "number of clients currently connected to api server",
	})

	gatewayStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "naisdevice",
		Subsystem: "apiserver",
		Name:      "gateway_status",
		Help:      "up/down status per gateway",
	}, []string{"gateway"})

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
		gatewayStatus,
		PrivilegedUsersPerGateway,
		DeviceConfigsReturned,
		GatewayConfigsReturned,
	)
}
