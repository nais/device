package main

import (
	"bytes"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nais/device/apiserver/kolide"

	"github.com/nais/device/apiserver/gatewayconfigurer"
	"github.com/nais/device/apiserver/jita"
	"github.com/nais/device/pkg/basicauth"
	"github.com/nais/device/pkg/pb"

	"github.com/golang-jwt/jwt"
	"github.com/nais/device/apiserver/auth"
	"github.com/nais/device/apiserver/azure/discovery"
	"github.com/nais/device/apiserver/azure/validate"
	"github.com/nais/device/apiserver/enroller"
	"github.com/nais/device/pkg/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nais/device/apiserver/api"
	"github.com/nais/device/apiserver/config"
	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

const (
	gatewayConfigSyncInterval = 1 * time.Minute
)

var (
	cfg = config.DefaultConfig()
)

func init() {
	flag.StringVar(&cfg.DbConnDSN, "db-connection-dsn", os.Getenv("DB_CONNECTION_DSN"), "database connection DSN")
	flag.StringVar(&cfg.JitaUsername, "jita-username", os.Getenv("JITA_USERNAME"), "jita username")
	flag.StringVar(&cfg.JitaPassword, "jita-password", os.Getenv("JITA_PASSWORD"), "jita password")
	flag.StringVar(&cfg.JitaUrl, "jita-url", os.Getenv("JITA_URL"), "jita URL")
	flag.StringVar(&cfg.BootstrapAPIURL, "bootstrap-api-url", "", "bootstrap API URL")
	flag.StringVar(&cfg.BootstrapApiCredentials, "bootstrap-api-credentials", os.Getenv("BOOTSTRAP_API_CREDENTIALS"), "bootstrap API credentials")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.PrometheusPublicKey, "prometheus-public-key", cfg.PrometheusPublicKey, "prometheus public key")
	flag.StringVar(&cfg.PrometheusTunnelIP, "prometheus-tunnel-ip", cfg.PrometheusTunnelIP, "prometheus tunnel ip")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "which log level to output")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.GRPCBindAddress, "grpc-bind-address", cfg.GRPCBindAddress, "Bind address for gRPC server")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "Path to configuration directory")
	flag.StringVar(&cfg.Endpoint, "endpoint", cfg.Endpoint, "public endpoint (ip:port)")
	flag.BoolVar(&cfg.DevMode, "development-mode", cfg.DevMode, "Development mode avoids setting up wireguard and fetching and validating AAD certificates")
	flag.StringVar(&cfg.Azure.DiscoveryURL, "azure-discovery-url", "", "Azure discovery url")
	flag.StringVar(&cfg.Azure.ClientID, "azure-client-id", "", "Azure app client id")
	flag.StringVar(&cfg.Azure.ClientSecret, "azure-client-secret", "", "Azure app client secret")
	flag.StringSliceVar(&cfg.CredentialEntries, "credential-entries", nil, "Comma-separated credentials on format: '<user>:<key>'")
	flag.StringVar(&cfg.GatewayConfigBucketName, "gateway-config-bucket-name", "gatewayconfig", "Name of bucket containing gateway config object")
	flag.StringVar(&cfg.GatewayConfigBucketObjectName, "gateway-config-bucket-object-name", "gatewayconfig.json", "Name of bucket object containing gateway config JSON")
	flag.StringVar(&cfg.KolideEventHandlerAddress, "kolide-event-handler-address", "", "address for kolide-event-handler grpc connection")
	flag.StringVar(&cfg.KolideEventHandlerToken, "kolide-event-handler-token", "", "token for kolide-event-handler grpc connection")
	flag.StringVar(&cfg.KolideApiToken, "kolide-api-token", "", "token used to communicate with the kolide api")

	flag.Parse()

	cfg.PrivateKeyPath = filepath.Join(cfg.ConfigDir, "private.key")
	cfg.WireGuardConfigPath = filepath.Join(cfg.ConfigDir, "wg0.conf")
	logger.Setup(cfg.LogLevel)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(cfg.KolideApiToken) == 0 {
		log.Warnf("no kolide api token provided, no device health updates will be performed")
	}

	if len(cfg.KolideEventHandlerAddress) > 0 && len(cfg.KolideEventHandlerToken) == 0 {
		log.Errorf("--kolide-event-handler-address is set, but --kolide-event-handler-token is not. aborting")
		return
	}

	api.InitializeMetrics()
	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	if err := setupInterface(); err != nil && !cfg.DevMode {
		log.Fatalf("Setting up WireGuard interface: %v", err)
	}

	var dbDriver string

	if cfg.DevMode {
		dbDriver = "postgres"
	} else {
		dbDriver = "cloudsqlpostgres"
	}

	db, err := database.New(cfg.DbConnDSN, dbDriver)
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

	updates := make(chan *database.Device, 64)

	kolideHandler := kolide.New(cfg.KolideApiToken, db, updates)

	go kolideHandler.Cron(ctx)

	if cfg.KolideEventHandlerAddress != "" {
		go kolideHandler.DeviceEventHandler(ctx, cfg.KolideEventHandlerAddress, cfg.KolideEventHandlerToken)
	}

	if len(cfg.BootstrapAPIURL) > 0 {
		parts := strings.Split(cfg.BootstrapApiCredentials, ":")
		username, password := parts[0], parts[1]

		en := enroller.Enroller{
			Client:             basicauth.Transport{Username: username, Password: password}.Client(),
			DB:                 db,
			BootstrapAPIURL:    cfg.BootstrapAPIURL,
			APIServerPublicKey: string(publicKey),
			APIServerEndpoint:  cfg.Endpoint,
		}

		go en.WatchDeviceEnrollments(ctx)
		go en.WatchGatewayEnrollments(ctx)
	}

	gwc := gatewayconfigurer.GatewayConfigurer{
		DB:           db,
		BucketReader: gatewayconfigurer.GoogleBucketReader{BucketName: cfg.GatewayConfigBucketName, BucketObjectName: cfg.GatewayConfigBucketObjectName},
		SyncInterval: gatewayConfigSyncInterval,
	}

	go gwc.SyncContinuously(ctx)

	go syncWireguardConfig(cfg.DbConnDSN, dbDriver, string(privateKey), cfg)

	apiConfig := api.Config{
		DB:       db,
		Jita:     jita.New(cfg.JitaUsername, cfg.JitaPassword, cfg.JitaUrl),
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

	grpcHandler := api.NewGRPCServer(db)
	grpcServer := grpc.NewServer()

	pb.RegisterAPIServerServer(grpcServer, grpcHandler)

	grpcListener, err := net.Listen("tcp", cfg.GRPCBindAddress)
	if err != nil {
		log.Fatalf("unable to set up gRPC server: %v", err)
	}

	// fixme: teardown/restart if this exits
	go grpcServer.Serve(grpcListener)

	go func() {
		for {
			device := <-updates
			sessionKey, err := sessions.SessionKeyFromDeviceID(device.ID)
			log.Infof("Pushing configuration for device %d, session key %s, error %s", device.ID, sessionKey, err)
			if err == nil {
				err = grpcHandler.SendDeviceConfiguration(context.TODO(), sessionKey)
			}
			if err != nil {
				log.Error(err)
			}
		}
	}()

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

func syncWireguardConfig(dbConnDSN, driver, privateKey string, conf config.Config) {
	db, err := database.New(dbConnDSN, driver)
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

func GenerateWGConfig(devices []database.Device, gateways []*pb.Gateway, privateKey string, conf config.Config) []byte {
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
		wgConfig += fmt.Sprintf(peerTemplate, gateway.Ip, gateway.PublicKey)
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
