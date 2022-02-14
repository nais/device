package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/pkg/apiserver/api"
	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/bucket"
	"github.com/nais/device/pkg/apiserver/config"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/enroller"
	"github.com/nais/device/pkg/apiserver/gatewayconfigurer"
	"github.com/nais/device/pkg/apiserver/jita"
	"github.com/nais/device/pkg/apiserver/kolide"
	apiserver_metrics "github.com/nais/device/pkg/apiserver/metrics"
	"github.com/nais/device/pkg/apiserver/wireguard"
	"github.com/nais/device/pkg/basicauth"
	"github.com/nais/device/pkg/logger"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
)

const (
	gatewayConfigSyncInterval = 1 * time.Minute
	WireGuardSyncInterval     = 10 * time.Second
	sendGatewayConfigTimeout  = 5 * time.Second
	sendDeviceUpdateTimeout   = 5 * time.Second
	shutdownGracePeriod       = 20 * time.Millisecond // time to allow server processes to finish their goroutines
)

var (
	cfg = config.DefaultConfig()
)

func init() {
	flag.StringVar(&cfg.DbConnDSN, "db-connection-dsn", cfg.DbConnDSN, "database connection DSN")
	flag.StringVar(&cfg.JitaUsername, "jita-username", cfg.JitaUsername, "jita username")
	flag.StringVar(&cfg.JitaPassword, "jita-password", cfg.JitaPassword, "jita password")
	flag.StringVar(&cfg.JitaUrl, "jita-url", cfg.JitaUrl, "jita URL")
	flag.StringVar(&cfg.BootstrapAPIURL, "bootstrap-api-url", cfg.BootstrapAPIURL, "bootstrap API URL")
	flag.StringVar(&cfg.BootstrapApiCredentials, "bootstrap-api-credentials", cfg.BootstrapApiCredentials, "bootstrap API credentials")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.PrometheusPublicKey, "prometheus-public-key", cfg.PrometheusPublicKey, "prometheus public key")
	flag.StringVar(&cfg.PrometheusTunnelIP, "prometheus-tunnel-ip", cfg.PrometheusTunnelIP, "prometheus tunnel ip")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.GRPCBindAddress, "grpc-bind-address", cfg.GRPCBindAddress, "Bind address for gRPC server")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "Path to configuration directory")
	flag.StringVar(&cfg.Endpoint, "endpoint", cfg.Endpoint, "public endpoint (ip:port)")
	flag.StringVar(&cfg.Azure.ClientID, "azure-client-id", cfg.Azure.ClientID, "Azure app client id")
	flag.StringVar(&cfg.Azure.Tenant, "azure-tenant", cfg.Azure.Tenant, "Azure tenant")
	flag.StringVar(&cfg.Azure.ClientSecret, "azure-client-secret", cfg.Azure.ClientSecret, "Azure app client secret")
	flag.StringSliceVar(&cfg.AdminCredentialEntries, "admin-credential-entries", cfg.AdminCredentialEntries, "Comma-separated credentials on format: '<user>:<key>'")
	flag.StringSliceVar(&cfg.PrometheusCredentialEntries, "prometheus-credential-entries", cfg.PrometheusCredentialEntries, "Comma-separated credentials on format: '<user>:<key>'")
	flag.StringVar(&cfg.GatewayConfigBucketName, "gateway-config-bucket-name", cfg.GatewayConfigBucketName, "Name of bucket containing gateway config object")
	flag.StringVar(&cfg.GatewayConfigBucketObjectName, "gateway-config-bucket-object-name", cfg.GatewayConfigBucketObjectName, "Name of bucket object containing gateway config JSON")
	flag.StringVar(&cfg.KolideEventHandlerAddress, "kolide-event-handler-address", cfg.KolideEventHandlerAddress, "address for kolide-event-handler grpc connection")
	flag.BoolVar(&cfg.KolideEventHandlerEnabled, "kolide-event-handler-enabled", cfg.KolideEventHandlerEnabled, "enable kolide event handler (incoming webhooks from kolide on device failures)")
	flag.StringVar(&cfg.KolideEventHandlerToken, "kolide-event-handler-token", cfg.KolideEventHandlerToken, "token for kolide-event-handler grpc connection")
	flag.StringVar(&cfg.KolideApiToken, "kolide-api-token", cfg.KolideApiToken, "token used to communicate with the kolide api")
	flag.BoolVar(&cfg.KolideSyncEnabled, "kolide-sync-enabled", cfg.KolideSyncEnabled, "enable kolide sync integration (looking for device failures)")
	flag.BoolVar(&cfg.DeviceAuthenticationEnabled, "device-authentication-enabled", cfg.DeviceAuthenticationEnabled, "enable authentication for nais devices (oauth2)")
	flag.BoolVar(&cfg.ControlPlaneAuthenticationEnabled, "control-plane-authentication-enabled", cfg.ControlPlaneAuthenticationEnabled, "enable authentication for control plane (api keys)")
	flag.BoolVar(&cfg.WireGuardEnabled, "wireguard-enabled", cfg.WireGuardEnabled, "enable WireGuard")
	flag.BoolVar(&cfg.CloudSQLProxyEnabled, "cloud-sql-proxy-enabled", cfg.CloudSQLProxyEnabled, "enable Google Cloud SQL proxy for database connection")

	flag.Parse()

	cfg.PrivateKeyPath = filepath.Join(cfg.ConfigDir, "private.key")
	cfg.WireGuardConfigPath = filepath.Join(cfg.ConfigDir, "wg0.conf")

	logger.Setup(cfg.LogLevel)
}

var errRequiredArgNotSet = errors.New("arg is required, but not set")

func main() {
	err := run()
	if err != nil {
		if errors.Is(err, errRequiredArgNotSet) {
			flag.Usage()
			log.Error(err)
		} else {
			log.Errorf("fatal error: %s", err)
		}

		os.Exit(1)
	} else {
		log.Info("naisdevice API server has shut down cleanly.")
	}
}

func run() error {
	var authenticator auth.Authenticator
	var adminAuthenticator auth.UsernamePasswordAuthenticator
	var gatewayAuthenticator auth.UsernamePasswordAuthenticator
	var prometheusAuthenticator auth.UsernamePasswordAuthenticator
	var wireguardPublicKey []byte

	err := envconfig.Process("APISERVER", &cfg)
	if err != nil {
		return fmt.Errorf("parse environment variables: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Infof("naisdevice API server %s starting up", version.Version)

	db, err := database.New(cfg.DbConnDSN, cfg.DatabaseDriver())
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}

	log.Infof("Loading user sessions from database...")

	sessions := auth.NewSessionStore(db)
	err = sessions.Warmup(ctx)
	if err != nil {
		return fmt.Errorf("warm session cache from database: %w", err)
	}

	if cfg.DeviceAuthenticationEnabled {
		log.Infof("Fetching Azure OIDC configuration...")
		err = cfg.Azure.FetchCertificates()
		if err != nil {
			return fmt.Errorf("fetch jwks: %w", err)
		}

		authenticator = auth.NewAuthenticator(cfg.Azure, db, sessions)
		log.Infof("Azure OIDC authenticator configured to authenticate device sessions.")

	} else {

		authenticator = auth.NewMockAuthenticator(sessions)
		log.Warnf("Device authentication DISABLED! Do not run this configuration in production!")
	}

	if cfg.WireGuardEnabled {
		log.Infof("Setting up WireGuard integration...")

		err = setupInterface()
		if err != nil {
			return fmt.Errorf("set up WireGuard interface: %w", err)
		}

		privateKey, err := ioutil.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("read WireGuard private key: %w", err)
		}

		wireguardPublicKey, err = generatePublicKey(privateKey, "wg")
		if err != nil {
			return fmt.Errorf("generate WireGuard public key: %w", err)
		}

		w := wireguard.New(cfg, db, string(privateKey))

		go SyncLoop(w)

		log.Infof("WireGuard successfully configured.")

	} else {
		log.Warnf("WireGuard integration DISABLED! Do not run this configuration in production!")
	}

	deviceUpdates := make(chan *pb.Device, 64)
	triggerGatewaySync := make(chan struct{}, 64)

	// TODO: remove when we've improved JITA
	// This triggers sync every 10 sec to let gateways know if someone has JITA'd
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				log.Info("Hack: triggering gateway-sync")
				triggerGatewaySync <- struct{}{}

			case <-ctx.Done():
				log.Infof("Stopped gateway-sync hack")
				return
			}
		}
	}()

	if cfg.KolideSyncEnabled {
		if len(cfg.KolideApiToken) == 0 {
			return fmt.Errorf("--kolide-api-token %w", errRequiredArgNotSet)
		}

		kolideHandler := kolide.New(cfg.KolideApiToken, db, deviceUpdates, triggerGatewaySync)

		go kolideHandler.Cron(ctx)
	}

	if cfg.KolideEventHandlerEnabled {
		if len(cfg.KolideApiToken) == 0 {
			return fmt.Errorf("--kolide-api-token %w", errRequiredArgNotSet)
		}
		if len(cfg.KolideEventHandlerAddress) == 0 {
			return fmt.Errorf("--kolide-event-handler-address %w", errRequiredArgNotSet)
		}
		if len(cfg.KolideEventHandlerToken) == 0 {
			return fmt.Errorf("--kolide-event-handler-token %w", errRequiredArgNotSet)
		}

		kolideHandler := kolide.New(cfg.KolideApiToken, db, deviceUpdates, triggerGatewaySync)
		go kolideHandler.DeviceEventHandler(ctx, cfg.KolideEventHandlerAddress, cfg.KolideEventHandlerToken)
	}

	if len(cfg.BootstrapAPIURL) > 0 {
		parts := strings.Split(cfg.BootstrapApiCredentials, ":")
		username, password := parts[0], parts[1]

		en := enroller.Enroller{
			Client:             basicauth.Transport{Username: username, Password: password}.Client(),
			DB:                 db,
			BootstrapAPIURL:    cfg.BootstrapAPIURL,
			APIServerPublicKey: string(wireguardPublicKey),
			APIServerEndpoint:  cfg.Endpoint,
		}

		go en.WatchDeviceEnrollments(ctx)
	}

	buck := bucket.NewClient(cfg.GatewayConfigBucketName, cfg.GatewayConfigBucketObjectName)

	gwc := gatewayconfigurer.GatewayConfigurer{
		DB:                 db,
		Bucket:             buck,
		SyncInterval:       gatewayConfigSyncInterval,
		TriggerGatewaySync: triggerGatewaySync,
	}

	jitaClient := jita.New(cfg.JitaUsername, cfg.JitaPassword, cfg.JitaUrl)

	go gwc.SyncContinuously(ctx)

	apiConfig := api.Config{
		DB:            db,
		Jita:          jitaClient,
		Authenticator: authenticator,
	}

	if cfg.ControlPlaneAuthenticationEnabled {
		apiConfig.APIKeys, err = config.Credentials(cfg.AdminCredentialEntries)
		if err != nil {
			return fmt.Errorf("parse admin credentials: %w", err)
		}

		if len(apiConfig.APIKeys) == 0 {
			return fmt.Errorf("control plane basic authentication enabled, but no admin credentials provided (try --admin-credential-entries)")
		}

		promauth, err := config.Credentials(cfg.PrometheusCredentialEntries)
		if err != nil {
			return fmt.Errorf("parse prometheus credentials: %w", err)
		}

		if len(promauth) == 0 {
			return fmt.Errorf("control plane basic authentication enabled, but no prometheus credentials provided (try --prometheus-credential-entries)")
		}

		adminAuthenticator = auth.NewAPIKeyAuthenticator(apiConfig.APIKeys)
		gatewayAuthenticator = auth.NewGatewayAuthenticator(db)
		prometheusAuthenticator = auth.NewAPIKeyAuthenticator(promauth)

		log.Warnf("Control plane authentication enabled.")

	} else {
		adminAuthenticator = auth.NewMockAPIKeyAuthenticator()
		gatewayAuthenticator = auth.NewMockAPIKeyAuthenticator()
		prometheusAuthenticator = auth.NewMockAPIKeyAuthenticator()

		log.Warnf("Control plane authentication DISABLED! Do not run this configuration in production!")
	}

	grpcHandler := api.NewGRPCServer(
		db,
		authenticator,
		adminAuthenticator,
		gatewayAuthenticator,
		prometheusAuthenticator,
		jitaClient,
		triggerGatewaySync,
	)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	)
	pb.RegisterAPIServerServer(grpcServer, grpcHandler)
	grpc_prometheus.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", cfg.GRPCBindAddress)
	if err != nil {
		return fmt.Errorf("unable to set up gRPC server: %w", err)
	}

	sendDeviceConfig := func(device *pb.Device) {
		ctx, cancel := context.WithTimeout(ctx, sendDeviceUpdateTimeout)
		defer cancel()

		session, err := sessions.CachedSessionFromDeviceID(device.Id)
		log.Debugf("Pushing configuration for device %d, error %s", device.Id, err)
		if err == nil {
			err = grpcHandler.SendDeviceConfiguration(ctx, session.GetKey())
		}
		if err != nil && !errors.Is(err, api.ErrNoSession) {
			// fixme: metrics
			log.Error(err)
		}
	}

	sendGatewayUpdates := func() {
		ctx, cancel := context.WithTimeout(ctx, sendGatewayConfigTimeout)
		defer cancel()

		err = grpcHandler.SendAllGatewayConfigurations(ctx)
		if err != nil {
			// fixme: metrics
			log.Error(err)
		}
	}

	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		err := apiserver_metrics.Serve(cfg.PrometheusAddr)
		if err != nil {
			log.Errorf("metrics server shut down with error; killing apiserver process: %s", err)
			cancel()
		}
	}()

	router := api.New(apiConfig)

	srv := &http.Server{
		Handler: router,
		Addr:    cfg.BindAddress,
	}

	go func() {
		log.Infof("Legacy HTTP API starting on %s", cfg.BindAddress)
		err := srv.ListenAndServe()
		cancel()
		switch err {
		case http.ErrServerClosed:
			log.Infof("HTTP server stopped successfully.")
		case nil:
		default:
			log.Errorf("HTTP server terminated with error: %s", err)
		}
	}()

	go func() {
		log.Infof("gRPC server starting on %s", cfg.GRPCBindAddress)
		err := grpcServer.Serve(grpcListener)
		if err != nil {
			log.Errorf("gRPC server exited with error: %s", err)
		}
		cancel()
	}()

	sigs := make(chan os.Signal)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	defer srv.Close()
	defer grpcServer.GracefulStop()

	for {
		select {
		case s := <-sigs:
			log.Warnf("Received signal %s", s)
			cancel()

		case <-ctx.Done():
			log.Warnf("Program context canceled; shutting down.")
			log.Warnf("Stopping legacy HTTP API...")
			err = srv.Close()
			if err != nil {
				log.Errorf("Shutdown: %s", err)
			}
			log.Warnf("Stopping gRPC API...")
			grpcServer.GracefulStop()
			time.Sleep(shutdownGracePeriod)
			return nil

		case device := <-deviceUpdates:
			sendDeviceConfig(device)

		case <-triggerGatewaySync:
			sendGatewayUpdates()
		}
	}
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

func SyncLoop(w wireguard.WireGuard) {
	log.Debugf("Starting config sync")

	ticker := time.NewTicker(WireGuardSyncInterval)
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), WireGuardSyncInterval)

		err := w.Sync(ctx)
		cancel()

		if err != nil {
			log.Errorf("syncing wg config: %s", err)
		}
	}
}
