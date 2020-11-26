package api

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

var (
	PrivilegedUsersPerGateway *prometheus.GaugeVec
)

func Serve(address string) {
	log.Infof("Prometheus serving metrics at %v", address)
	_ = http.ListenAndServe(address, promhttp.Handler())
}

func InitializeMetrics(name string) {

	PrivilegedUsersPerGateway = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "privileged_users",
		Help:      "privileged users per gateway",
		Namespace: "naisdevice",
		Subsystem: "apiserver",
	}, []string{"gateway"})

	prometheus.MustRegister(PrivilegedUsersPerGateway)

}
