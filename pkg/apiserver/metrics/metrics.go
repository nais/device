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

	gatewayStatus      *prometheus.GaugeVec
	gatewayConnections = make(map[string]bool)
)

func Serve(address string) error {
	return http.ListenAndServe(address, promhttp.Handler())
}

func SetConnectedGateways(gateways []string) {
	for k := range gatewayConnections {
		gatewayConnections[k] = false
	}
	for _, k := range gateways {
		gatewayConnections[k] = true
	}
	for k := range gatewayConnections {
		i := 0.0
		if gatewayConnections[k] {
			i = 1.0
		}
		gatewayStatus.With(prometheus.Labels{
			"gateway": k,
		}).Set(i)
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
