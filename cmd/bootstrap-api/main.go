package main

import (
	"crypto/x509"
	"github.com/go-chi/chi"
	chi_middleware "github.com/go-chi/chi/middleware"
	"github.com/nais/device/pkg/logger"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

type CertificateList []*x509.Certificate

type Azure struct {
	DiscoveryURL string
	ClientID     string
}

type Config struct {
	BindAddress         string
	Azure               Azure
	PrometheusAddr      string
	PrometheusPublicKey string
	PrometheusTunnelIP  string
	CredentialEntries   []string
	LogLevel            string
}

var cfg = &Config{
	Azure: Azure{
		ClientID:     "",
		DiscoveryURL: "",
	},
	CredentialEntries: nil,
	BindAddress:       ":8080",
	PrometheusAddr:    ":3000",
	LogLevel:          "info",
}

var deviceEnrollments ActiveDeviceEnrollments
var gatewayEnrollments ActiveGatewayEnrollments
const TokenHeaderKey = "x-naisdevice-gateway-token"

func init() {
	logger.Setup(cfg.LogLevel)

	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.Azure.DiscoveryURL, "azure-discovery-url", "", "Azure discovery url")
	flag.StringVar(&cfg.Azure.ClientID, "azure-client-id", "", "Azure app client id")
	flag.StringSliceVar(&cfg.CredentialEntries, "credential-entries", nil, "Comma-separated credentials on format: '<user>:<key>'")

	flag.Parse()

	deviceEnrollments.init()
	gatewayEnrollments.init()
}

func main() {

	parts := strings.Split(cfg.CredentialEntries[0], ":")
	credentialEntries := map[string]string{
		parts[0]: parts[1],
	}

	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	jwtValidator, err := createJWTValidator(cfg.Azure)
	if err != nil {
		log.Fatalf("Creating JWT validator: %v", err)
	}
	r := Api(credentialEntries, TokenValidatorMiddleware(jwtValidator))

	log.Info("running @", cfg.BindAddress)
	log.Info(http.ListenAndServe(cfg.BindAddress, r))
}

func Api(apiserverCredentialEntries map[string]string, azureValidator func(next http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Get("/isalive", func(w http.ResponseWriter, r *http.Request) {
		return
	})

	r.Route("/api/v1", func(r chi.Router) {
		// device calls
		r.Group(func(r chi.Router) {
			r.Use(azureValidator)
			r.Post("/deviceinfo", postDeviceInfo)
			r.Get("/bootstrapconfig/{serial}", getBootstrapConfig)
		})

		// apiserver calls
		r.Group(func(r chi.Router) {
			r.Use(chi_middleware.BasicAuth("naisdevice", apiserverCredentialEntries))
			r.Get("/deviceinfo", getDeviceInfos)
			r.Post("/bootstrapconfig/{serial}", postBootstrapConfig)
		})

		// gateway calls
		r.Group(func(r chi.Router) {
			r.Use(TokenAuth)
			r.Post("/gatewayinfo", postGatewayInfo)
			r.Get("/gatewayconfig", getGatewayConfig)
		})

		// apiserver calls
		r.Group(func(r chi.Router) {
			r.Use(chi_middleware.BasicAuth("naisdevice", apiserverCredentialEntries))
			r.Get("/gatewayinfo", getGatewayInfo)
			r.Post("/gatewayconfig", postGatewayConfig)
			r.Post("/token", postToken)
		})
	})

	return r
}

func TokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(TokenHeaderKey)

		if !gatewayEnrollments.hasToken(token) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
