package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nais/device/apiserver/azure/discovery"
	"github.com/nais/device/apiserver/azure/validate"
	"github.com/nais/device/apiserver/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nais/device/apiserver/api"
	"github.com/nais/device/apiserver/config"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/apiserver/slack"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = config.DefaultConfig()
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	flag.StringVar(&cfg.DbConnURI, "db-connection-uri", os.Getenv("DB_CONNECTION_URI"), "database connection URI (DSN)")
	flag.StringVar(&cfg.SlackToken, "slack-token", os.Getenv("SLACK_TOKEN"), "Slack token")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.PrometheusPublicKey, "prometheus-public-key", cfg.PrometheusPublicKey, "prometheus public key")
	flag.StringVar(&cfg.PrometheusTunnelIP, "prometheus-tunnel-ip", cfg.PrometheusTunnelIP, "prometheus tunnel ip")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "Path to configuration directory")
	flag.StringVar(&cfg.Endpoint, "endpoint", cfg.Endpoint, "public endpoint (ip:port)")
	flag.BoolVar(&cfg.DevMode, "development-mode", cfg.DevMode, "Development mode avoids setting up wireguard and fetching and validating AAD certificates")
	flag.StringVar(&cfg.Azure.DiscoveryURL, "azure-discovery-url", "", "Azure discovery url")
	flag.StringVar(&cfg.Azure.ClientID, "azure-client-id", "", "Azure app client id")

	flag.Parse()

	cfg.PrivateKeyPath = filepath.Join(cfg.ConfigDir, "private.key")
	cfg.WireGuardConfigPath = filepath.Join(cfg.ConfigDir, "wg0.conf")

	if err := cfg.Valid(); err != nil && !cfg.DevMode {
		fmt.Printf("Validating config: %v", err)
		os.Exit(1)
	}
}

func main() {
	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	if !cfg.DevMode {
		if err := setupInterface(); err != nil {
			log.Fatalf("Setting up WireGuard interface: %v", err)
		}
	} else {
		log.Infof("DevMode: skipping interface setup")
	}

	db, err := database.New(cfg.DbConnURI)
	if err != nil {
		log.Fatalf("Instantiating database: %s", err)
	}

	privateKey, err := ioutil.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		log.Fatalf("Reading private key: %v", err)
	}

	publicKey, err := generatePublicKey(privateKey, "wg")
	if err != nil {
		log.Fatalf("Generating public key: %v", err)
	}

	certificates, err := discovery.FetchCertificates(cfg.Azure)
	if err != nil && !cfg.DevMode {
		log.Fatalf("Retrieving azure ad certificates for token validation: %v", err)
	}

	jwtValidator := validate.JWTValidator(certificates, cfg.Azure.ClientID)

	if len(cfg.SlackToken) > 0 {
		slackBot := slack.New(cfg.SlackToken, cfg.Endpoint, db, string(publicKey), jwtValidator)
		go slackBot.Handler()
	}

	if !cfg.DevMode {
		go syncWireguardConfig(cfg.DbConnURI, string(privateKey), cfg)
	}

	apiConfig := api.Config{
		DB: db,
	}

	if !cfg.DevMode {
		apiConfig.OAuthKeyValidatorMiddleware = middleware.TokenValidatorMiddleware(jwtValidator)
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

	return out, nil
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

	for c := time.Tick(10 * time.Second); ; <-c {
		log.Info("Syncing config")
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
		}

		if b, err := exec.Command("wg", "syncconf", "wg0", conf.WireGuardConfigPath).Output(); err != nil {
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
