package main

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nais/device/pkg/azure"
	bootstrap_api "github.com/nais/device/pkg/bootstrap-api"
	"github.com/nais/device/pkg/logger"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

const SecretSyncInterval = 10 * time.Second

type Config struct {
	BindAddress         string
	Azure               *azure.Azure
	PrometheusAddr      string
	PrometheusPublicKey string
	PrometheusTunnelIP  string
	CredentialEntries   []string
	LogLevel            string
	AzureAuthEnabled    bool
}

var cfg = &Config{
	Azure:             &azure.Azure{},
	CredentialEntries: nil,
	BindAddress:       ":8080",
	PrometheusAddr:    ":3000",
	LogLevel:          "info",
}

func init() {
	logger.Setup(cfg.LogLevel)

	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.BoolVar(&cfg.AzureAuthEnabled, "azure-auth-enabled", false, "Azure auth enabled")
	flag.StringVar(&cfg.Azure.ClientID, "azure-client-id", "6e45010d-2637-4a40-b91d-d4cbb451fb57", "Azure app client id")
	flag.StringVar(&cfg.Azure.Tenant, "azure-tenant", "62366534-1ec3-4962-8869-9b5535279d0b", "Azure tenant")
	flag.StringSliceVar(&cfg.CredentialEntries, "credential-entries", nil, "Comma-separated credentials on format: '<user>:<key>'")

	flag.Parse()
}

func main() {
	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	var tokenValidator func(next http.Handler) http.Handler
	if cfg.AzureAuthEnabled {
		err := cfg.Azure.FetchCertificates()
		if err != nil {
			log.Fatalf("fetch azure certs: %s", err)
		}

		tokenValidator = cfg.Azure.TokenValidatorMiddleware()
	} else {
		log.Warnf("AUTH DISABLED, this should NOT run in production")
		tokenValidator = mockTokenValidator()
	}

	apiserverCredentials, err := bootstrap_api.Credentials(cfg.CredentialEntries)
	if err != nil {
		log.Fatalf("Reading basic auth credentials: %v", err)
	}

	apiLogger := log.WithField("component", "api")
	api := bootstrap_api.NewApi(apiserverCredentials, tokenValidator, apiLogger)
	router := api.Router()

	log.Info("running @ ", cfg.BindAddress)
	log.Info(http.ListenAndServe(cfg.BindAddress, router))
}

func mockTokenValidator() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w,
				r.WithContext(
					context.WithValue(
						r.Context(),
						"preferred_username",
						"username@mock.dev",
					),
				),
			)
		})
	}
}
