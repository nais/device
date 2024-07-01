package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "naisdevice"
	subsystem = "apiserver"
)

var (
	DeviceConfigsReturned     *prometheus.CounterVec
	DevicesConnected          prometheus.Gauge
	GatewayConfigsReturned    *prometheus.CounterVec
	PrivilegedUsersPerGateway *prometheus.GaugeVec
	LoginRequests             *prometheus.CounterVec

	deviceStreamsEnded prometheus.CounterVec
	gatewayStatus      *prometheus.GaugeVec
	kolideStatusCodes  *prometheus.CounterVec
)

func Serve(address string) error {
	return http.ListenAndServe(address, promhttp.Handler())
}

func SetGatewayConnected(name string, connected bool) {
	labels := prometheus.Labels{"gateway": name}
	if connected {
		gatewayStatus.With(labels).Set(1.0)
	} else {
		gatewayStatus.With(labels).Set(0.0)
	}
}

func IncKolideStatusCode(code int) {
	kolideStatusCodes.WithLabelValues(strconv.Itoa(code)).Inc()
}

func IncDeviceStreamsEnded(reason string) {
	deviceStreamsEnded.WithLabelValues(reason).Inc()
}

func init() {
	DevicesConnected = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "devices_connected",
		Help:      "number of clients currently connected to api server",
	})

	gatewayStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "gateway_status",
		Help:      "up/down status per gateway",
	}, []string{"gateway"})

	PrivilegedUsersPerGateway = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "privileged_users",
		Help:      "privileged users per gateway",
	}, []string{"gateway"})

	DeviceConfigsReturned = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "user_configs_returned",
		Help:      "Total number of configs returned to device since apiserver started",
	}, []string{"serial", "username"})

	GatewayConfigsReturned = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "gateway_configs_returned",
		Help:      "Total number of configs returned to gateway since apiserver started",
	}, []string{"gateway"})

	LoginRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "device_login_requests",
		Help:      "Device logins with agent version",
	}, []string{"version"})

	kolideStatusCodes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "kolide_status_codes",
		Help:      "Kolide status codes from API",
	}, []string{"code"})

	deviceStreamsEnded = *prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "device_streams_ended",
		Help:      "Device streams ending with reason label",
	}, []string{"reason"})

	prometheus.MustRegister(
		DevicesConnected,
		gatewayStatus,
		PrivilegedUsersPerGateway,
		DeviceConfigsReturned,
		GatewayConfigsReturned,
		LoginRequests,
		kolideStatusCodes,
		deviceStreamsEnded,
	)
}
