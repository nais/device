package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/nais/device/apiserver/auth"
	"github.com/nais/device/apiserver/azure/discovery"
	"github.com/nais/device/apiserver/azure/validate"
	"github.com/nais/device/apiserver/bootstrapper"
	"github.com/nais/device/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nais/device/apiserver/api"
	"github.com/nais/device/apiserver/config"
	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = config.DefaultConfig()
)

func init() {
	flag.StringVar(&cfg.DbConnURI, "db-connection-uri", os.Getenv("DB_CONNECTION_URI"), "database connection URI (DSN)")
	flag.StringVar(&cfg.BootstrapApiURL, "bootstrap-api-url", "", "bootstrap API URL")
	flag.StringVar(&cfg.BootstrapApiCredentials, "bootstrap-api-credentials", os.Getenv("BOOTSTRAP_API_CREDENTIALS"), "bootstrap API credentials")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.PrometheusPublicKey, "prometheus-public-key", cfg.PrometheusPublicKey, "prometheus public key")
	flag.StringVar(&cfg.PrometheusTunnelIP, "prometheus-tunnel-ip", cfg.PrometheusTunnelIP, "prometheus tunnel ip")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "Path to configuration directory")
	flag.StringVar(&cfg.Endpoint, "endpoint", cfg.Endpoint, "public endpoint (ip:port)")
	flag.BoolVar(&cfg.DevMode, "development-mode", cfg.DevMode, "Development mode avoids setting up wireguard and fetching and validating AAD certificates")
	flag.StringVar(&cfg.Azure.DiscoveryURL, "azure-discovery-url", "", "Azure discovery url")
	flag.StringVar(&cfg.Azure.ClientID, "azure-client-id", "", "Azure app client id")
	flag.StringVar(&cfg.Azure.ClientSecret, "azure-client-secret", "", "Azure app client secret")
	flag.StringSliceVar(&cfg.CredentialEntries, "credential-entries", nil, "Comma-separated credentials on format: '<user>:<key>'")

	flag.Parse()

	cfg.PrivateKeyPath = filepath.Join(cfg.ConfigDir, "private.key")
	cfg.WireGuardConfigPath = filepath.Join(cfg.ConfigDir, "wg0.conf")
	logger.Setup(cfg.LogLevel)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	if err := setupInterface(); err != nil && !cfg.DevMode {
		log.Fatalf("Setting up WireGuard interface: %v", err)
	}

	db, err := database.New(cfg.DbConnURI)
	if err != nil {
		log.Fatalf("Instantiating database: %s", err)
	}

	tokenValidator, err := createJWTValidator(cfg)
	if err != nil {
		log.Fatalf("creating JWT validator: %v", err)
	}

	sessions, err := auth.New(cfg, tokenValidator, db)
	if err != nil {
		log.Fatalf("Instantiating sessions: %s", err)
	}

	privateKey, err := ioutil.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		log.Fatalf("Reading private key: %v", err)
	}

	publicKey, err := generatePublicKey(privateKey, "wg")
	if err != nil {
		log.Fatalf("Generating public key: %v", err)
	}

	if len(cfg.BootstrapApiURL) > 0 {
		go bootstrapper.WatchEnrollments(ctx, db, cfg.BootstrapApiURL, cfg.BootstrapApiCredentials, publicKey, cfg.Endpoint)
	}

	go syncWireguardConfig(cfg.DbConnURI, string(privateKey), cfg)

	apiConfig := api.Config{
		DB:       db,
		Sessions: sessions,
	}

	apiConfig.APIKeys, err = cfg.Credentials()
	if err != nil {
		log.Fatalf("Getting credentials: %v", err)
	}

	if !cfg.DevMode {
		if apiConfig.APIKeys == nil {
			log.Fatalf("No credentials provided for basic auth")
		}
	}

	router := api.New(apiConfig)

	fmt.Println("running @", cfg.BindAddress)
	fmt.Println(http.ListenAndServe(cfg.BindAddress, router))
}

func generatePublicKey(privateKey []byte, wireGuardPath string) ([]byte, error) {
	cmd := exec.Command(wireGuardPath, "pubkey")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("opening stdin pipe to wg genkey: %w", err)
	}

	_, err = stdin.Write(privateKey)
	if err != nil {
		return nil, fmt.Errorf("writing to wg genkey stdin pipe: %w", err)
	}

	if err = stdin.Close(); err != nil {
		return nil, fmt.Errorf("closing stdin %w", err)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("executing command: %v: %w: %v", cmd, err, string(out))
	}

	return bytes.TrimSuffix(out, []byte("\n")), nil
}

func setupInterface() error {
	if err := exec.Command("ip", "link", "del", "wg0").Run(); err != nil {
		log.Infof("Pre-deleting WireGuard interface (ok if this fails): %v", err)
	}

	run := func(commands [][]string) error {
		for _, s := range commands {
			cmd := exec.Command(s[0], s[1:]...)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("running %v: %w: %v", cmd, err, string(out))
			} else {
				fmt.Printf("%v: %v\n", cmd, string(out))
			}
		}
		return nil
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", "wg0", "type", "wireguard"},
		{"ip", "link", "set", "wg0", "mtu", "1360"},
		{"ip", "address", "add", "dev", "wg0", "10.255.240.1/21"},
		{"ip", "link", "set", "wg0", "up"},
	}

	return run(commands)
}

func syncWireguardConfig(dbConnURI, privateKey string, conf config.Config) {
	db, err := database.New(dbConnURI)
	if err != nil {
		log.Fatalf("Instantiating database: %v", err)
	}

	log.Info("Starting config sync")
	for c := time.Tick(10 * time.Second); ; <-c {
		log.Debug("Synchronizing configuration")
		devices, err := db.ReadDevices()
		if err != nil {
			log.Errorf("Reading devices from database: %v", err)
		}

		gateways, err := db.ReadGateways()
		if err != nil {
			log.Errorf("Reading gateways from database: %v", err)
		}

		wgConfigContent := GenerateWGConfig(devices, gateways, privateKey, cfg)

		if err := ioutil.WriteFile(conf.WireGuardConfigPath, wgConfigContent, 0600); err != nil {
			log.Errorf("Writing WireGuard config to disk: %v", err)
		} else {
			log.Debugf("Successfully wrote WireGuard config to: %v", conf.WireGuardConfigPath)
		}

		syncConf := exec.Command("wg", "syncconf", "wg0", conf.WireGuardConfigPath)
		if cfg.DevMode {
			log.Infof("DevMode: skip running %v", syncConf)
		} else if b, err := syncConf.CombinedOutput(); err != nil {
			log.Errorf("Synchronizing WireGuard config: %v: %v", err, string(b))
		}
	}
}

func GenerateWGConfig(devices []database.Device, gateways []database.Gateway, privateKey string, conf config.Config) []byte {
	interfaceTemplate := `[Interface]
PrivateKey = %s
ListenPort = 51820

`

	wgConfig := fmt.Sprintf(interfaceTemplate, strings.TrimSuffix(privateKey, "\n"))

	peerTemplate := `[Peer]
AllowedIPs = %s/32
PublicKey = %s
`
	wgConfig += fmt.Sprintf(peerTemplate, conf.PrometheusTunnelIP, conf.PrometheusPublicKey)

	for _, device := range devices {
		wgConfig += fmt.Sprintf(peerTemplate, device.IP, device.PublicKey)
	}

	for _, gateway := range gateways {
		wgConfig += fmt.Sprintf(peerTemplate, gateway.IP, gateway.PublicKey)
	}

	return []byte(wgConfig)
}

func createJWTValidator(conf config.Config) (jwt.Keyfunc, error) {
	if conf.DevMode {
		return func(token *jwt.Token) (interface{}, error) {
			return []byte("never_used"), nil
		}, nil
	}

	if len(conf.Azure.ClientID) == 0 || len(conf.Azure.DiscoveryURL) == 0 {
		return nil, fmt.Errorf("missing required azure configuration")
	}

	certificates, err := discovery.FetchCertificates(conf.Azure)
	if err != nil {
		return nil, fmt.Errorf("retrieving azure ad certificates for token validation: %v", err)
	}

	return validate.JWTValidator(certificates, conf.Azure.ClientID), nil
}
