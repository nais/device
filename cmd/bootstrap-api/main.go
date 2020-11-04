package main

import (
	"github.com/nais/device/bootstrap-api"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/secretmanager"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

const SecretSyncInterval = 10 * time.Second

type Config struct {
	BindAddress            string
	Azure                  bootstrap_api.Azure
	PrometheusAddr         string
	PrometheusPublicKey    string
	PrometheusTunnelIP     string
	CredentialEntries      []string
	LogLevel               string
	SecretManagerProjectID string
}

var cfg = &Config{
	Azure: bootstrap_api.Azure{
		ClientID:     "",
		DiscoveryURL: "",
	},
	CredentialEntries: nil,
	BindAddress:       ":8080",
	PrometheusAddr:    ":3000",
	LogLevel:          "info",
}

func init() {
	logger.Setup(cfg.LogLevel)

	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.Azure.DiscoveryURL, "azure-discovery-url", "", "Azure discovery url")
	flag.StringVar(&cfg.Azure.ClientID, "azure-client-id", "", "Azure app client id")
	flag.StringVar(&cfg.SecretManagerProjectID, "secret-manager-project-id", "nais-device", "Secret Manager Project ID")
	flag.StringSliceVar(&cfg.CredentialEntries, "credential-entries", nil, "Comma-separated credentials on format: '<user>:<key>'")

	flag.Parse()
}

func main() {
	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	jwtValidator, err := bootstrap_api.CreateJWTValidator(cfg.Azure)
	if err != nil {
		log.Fatalf("Creating JWT validator: %v", err)
	}

	apiserverCredentials, err := bootstrap_api.Credentials(cfg.CredentialEntries)
	if err != nil {
		log.Fatalf("Reading basic auth credentials: %v", err)
	}

	sm, err := secretmanager.New(cfg.SecretManagerProjectID)
	if err != nil {
		log.Fatalf("instantiating secret manager: %v", err)
	}

	tokenValidator := bootstrap_api.TokenValidatorMiddleware(jwtValidator)

	api := bootstrap_api.NewApi(apiserverCredentials, tokenValidator, sm, SecretSyncInterval)

	log.Info("running @", cfg.BindAddress)
	log.Info(http.ListenAndServe(cfg.BindAddress, api))
}
