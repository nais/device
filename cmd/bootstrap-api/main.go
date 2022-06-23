package main

import (
	"net/http"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nais/device/pkg/auth"
	bootstrap_api "github.com/nais/device/pkg/bootstrap-api"
	"github.com/nais/device/pkg/logger"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

const SecretSyncInterval = 10 * time.Second

type Config struct {
	BindAddress         string
	Azure               *auth.Azure
	Google              *auth.Google
	PrometheusAddr      string
	PrometheusPublicKey string
	PrometheusTunnelIP  string
	CredentialEntries   []string
	LogLevel            string
	AzureAuthEnabled    bool
	GoogleAuthEnabled   bool
}

func DefaultConfig() *Config {
	return &Config{
		Azure: &auth.Azure{
			ClientID: "6e45010d-2637-4a40-b91d-d4cbb451fb57",
			Tenant:   "62366534-1ec3-4962-8869-9b5535279d0b",
		},
		Google: &auth.Google{
			ClientID: "955023559628-g51n36t4icbd6lq7ils4r0ol9oo8kpk0.apps.googleusercontent.com",
		},
		CredentialEntries: nil,
		BindAddress:       ":8080",
		PrometheusAddr:    ":3000",
		LogLevel:          "info",
	}
}

func parseFlags(cfg *Config) {
	logger.Setup(cfg.LogLevel)

	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")

	flag.BoolVar(&cfg.AzureAuthEnabled, "azure-auth-enabled", cfg.AzureAuthEnabled, "Azure auth enabled")
	flag.StringVar(&cfg.Azure.ClientID, "azure-client-id", cfg.Azure.ClientID, "Azure app client id")
	flag.StringVar(&cfg.Azure.Tenant, "azure-tenant", cfg.Azure.Tenant, "Azure tenant")

	flag.BoolVar(&cfg.GoogleAuthEnabled, "google-auth-enabled", cfg.GoogleAuthEnabled, "Google auth enabled")
	flag.StringVar(&cfg.Google.ClientID, "google-client-id", cfg.Google.ClientID, "Google credential client id")
	flag.StringSliceVar(&cfg.Google.AllowedDomains, "google-allowed-domains", cfg.Google.AllowedDomains, "Comma-separated allowed domains on format: 'nais.io,partner.dev'")

	flag.StringSliceVar(&cfg.CredentialEntries, "credential-entries", cfg.CredentialEntries, "Comma-separated credentials on format: '<user>:<key>'")

	flag.Parse()
}

func main() {
	cfg := DefaultConfig()
	err := envconfig.Process("BOOTSTRAP_API", cfg)
	if err != nil {
		log.Fatalf("process envconfig: %s", err)
	}
	parseFlags(cfg)

	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	var tokenValidator auth.TokenValidator
	if cfg.AzureAuthEnabled && cfg.GoogleAuthEnabled {
		log.Fatal("Both Google and Azure auth enabled - pick one")
	}

	if cfg.AzureAuthEnabled {
		err := cfg.Azure.SetupJwkSetAutoRefresh()
		if err != nil {
			log.Fatalf("fetch Azure certs: %s", err)
		}

		tokenValidator = cfg.Azure.TokenValidatorMiddleware()
	} else if cfg.GoogleAuthEnabled {
		err := cfg.Google.SetupJwkSetAutoRefresh()
		if err != nil {
			log.Fatalf("fetch Google certs: %s", err)
		}

		tokenValidator = cfg.Google.TokenValidatorMiddleware()
	} else {
		log.Warnf("AUTH DISABLED, this should NOT run in production")
		tokenValidator = auth.MockTokenValidator()
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
