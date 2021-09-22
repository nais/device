package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nais/device/pkg/azure"
	bootstrap_api "github.com/nais/device/pkg/bootstrap-api"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/secretmanager"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

const SecretSyncInterval = 10 * time.Second

type Config struct {
	BindAddress            string
	Azure                  azure.Azure
	PrometheusAddr         string
	PrometheusPublicKey    string
	PrometheusTunnelIP     string
	CredentialEntries      []string
	LogLevel               string
	SecretManagerProjectID string
	DevMode                bool
}

var cfg = &Config{
	Azure: azure.Azure{
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
	flag.BoolVar(&cfg.DevMode, "development-mode", cfg.DevMode, "Development mode avoids setting up wireguard and fetching and validating AAD certificates")

	flag.Parse()
}

func main() {
	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	devMode := true
	jwtValidator, err := azure.CreateJWTValidator(cfg.Azure)
	if err != nil {
		if !devMode {
			log.Fatalf("Creating JWT validator: %v", err)
		}
	}

	apiserverCredentials, err := bootstrap_api.Credentials(cfg.CredentialEntries)
	if err != nil {
		log.Fatalf("Reading basic auth credentials: %v", err)
	}

	sm, err := secretmanager.New(cfg.SecretManagerProjectID)
	if err != nil {
		log.Fatalf("instantiating secret manager: %v", err)
	}

	tokenValidator := azure.TokenValidatorMiddleware(jwtValidator)

	api := bootstrap_api.NewApi(apiserverCredentials, tokenValidator, sm)
	router := api.Router()
	stop := make(chan struct{}, 1)
	go api.SyncEnrollmentSecretsLoop(SecretSyncInterval, stop)

	log.Info("running @ ", cfg.BindAddress)
	log.Info(http.ListenAndServe(cfg.BindAddress, router))
}
