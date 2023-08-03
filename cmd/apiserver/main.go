package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	wg "github.com/nais/device/pkg/wireguard"
	kolidepb "github.com/nais/kolide-event-handler/pkg/pb"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"
)

const (
	gatewayConfigSyncInterval = 1 * time.Minute
	WireGuardSyncInterval     = 20 * time.Second
)

var cfg = config.DefaultConfig()

func init() {
	flag.StringVar(&cfg.DBPath, "db-path", cfg.DBPath, "database file path")
	flag.StringVar(&cfg.JitaUsername, "jita-username", cfg.JitaUsername, "jita username")
	flag.StringVar(&cfg.JitaPassword, "jita-password", cfg.JitaPassword, "jita password")
	flag.StringVar(&cfg.JitaUrl, "jita-url", cfg.JitaUrl, "jita URL")
	flag.BoolVar(&cfg.JitaEnabled, "jita-enabled", cfg.JitaEnabled, "enable jita-synchronization")
	flag.StringVar(&cfg.BootstrapAPIURL, "bootstrap-api-url", cfg.BootstrapAPIURL, "bootstrap API URL")
	flag.StringVar(&cfg.BootstrapApiCredentials, "bootstrap-api-credentials", cfg.BootstrapApiCredentials, "bootstrap API credentials")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.PrometheusPublicKey, "prometheus-public-key", cfg.PrometheusPublicKey, "prometheus public key")
	flag.StringVar(&cfg.PrometheusTunnelIP, "prometheus-tunnel-ip", cfg.PrometheusTunnelIP, "prometheus tunnel ip")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "which log level to output")
	flag.StringVar(&cfg.BindAddress, "bind-address", cfg.BindAddress, "Bind address")
	flag.StringVar(&cfg.GRPCBindAddress, "grpc-bind-address", cfg.GRPCBindAddress, "Bind address for gRPC server")
	flag.StringVar(&cfg.Endpoint, "endpoint", cfg.Endpoint, "public endpoint (ip:port)")
	flag.StringVar(&cfg.Azure.ClientID, "azure-client-id", cfg.Azure.ClientID, "Azure app client id")
	flag.StringVar(&cfg.Azure.Tenant, "azure-tenant", cfg.Azure.Tenant, "Azure tenant")
	flag.StringVar(&cfg.Azure.ClientSecret, "azure-client-secret", cfg.Azure.ClientSecret, "Azure app client secret")
	flag.StringVar(&cfg.Google.ClientID, "google-client-id", cfg.Google.ClientID, "Google credential client id")
	flag.StringSliceVar(&cfg.Google.AllowedDomains, "google-allowed-domains", cfg.Google.AllowedDomains, "Google allowed domains: comma separated list")
	flag.StringSliceVar(&cfg.AdminCredentialEntries, "admin-credential-entries", cfg.AdminCredentialEntries, "Comma-separated credentials on format: '<user>:<key>'")
	flag.StringSliceVar(&cfg.PrometheusCredentialEntries, "prometheus-credential-entries", cfg.PrometheusCredentialEntries, "Comma-separated credentials on format: '<user>:<key>'")
	flag.StringVar(&cfg.GatewayConfigBucketName, "gateway-config-bucket-name", cfg.GatewayConfigBucketName, "Name of bucket containing gateway config object")
	flag.StringVar(&cfg.GatewayConfigBucketObjectName, "gateway-config-bucket-object-name", cfg.GatewayConfigBucketObjectName, "Name of bucket object containing gateway config JSON")
	flag.StringVar(&cfg.KolideEventHandlerAddress, "kolide-event-handler-address", cfg.KolideEventHandlerAddress, "address for kolide-event-handler grpc connection")
	flag.BoolVar(&cfg.KolideEventHandlerEnabled, "kolide-event-handler-enabled", cfg.KolideEventHandlerEnabled, "enable kolide event handler (incoming webhooks from kolide on device failures)")
	flag.BoolVar(&cfg.KolideEventHandlerSecure, "kolide-event-handler-secure", cfg.KolideEventHandlerSecure, "require TLS and authentication when talking to Kolide event handler")
	flag.StringVar(&cfg.KolideEventHandlerToken, "kolide-event-handler-token", cfg.KolideEventHandlerToken, "token for kolide-event-handler grpc connection")
	flag.StringVar(&cfg.DeviceAuthenticationProvider, "device-authentication-provider", cfg.DeviceAuthenticationProvider, "set device authentication provider")
	flag.BoolVar(&cfg.ControlPlaneAuthenticationEnabled, "control-plane-authentication-enabled", cfg.ControlPlaneAuthenticationEnabled, "enable authentication for control plane (api keys)")
	flag.BoolVar(&cfg.WireGuardEnabled, "wireguard-enabled", cfg.WireGuardEnabled, "enable WireGuard")
	flag.StringVar(&cfg.WireGuardIP, "wireguard-ip", cfg.WireGuardIP, "WireGuard ip")
	flag.StringVar(&cfg.WireGuardNetworkAddress, "wireguard-network-address", cfg.WireGuardNetworkAddress, "WireGuard network-address")
	flag.StringVar(&cfg.WireGuardPrivateKeyPath, "wireguard-private-key-path", cfg.WireGuardPrivateKeyPath, "WireGuard private key path")
	flag.StringVar(&cfg.GatewayConfigurer, "gateway-configurer", cfg.GatewayConfigurer, "which method to use for fetching gateway config (metadata or bucket)")
	flag.BoolVar(&cfg.AutoEnrollEnabled, "auto-enroll-enabled", cfg.AutoEnrollEnabled, "enable auto enroll support using pub/sub")

	flag.Parse()
}

var errRequiredArgNotSet = errors.New("arg is required, but not set")

func main() {
	// sets up default logger
	logger.Setup(cfg.LogLevel)

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

	err := envconfig.Process("APISERVER", &cfg)
	if err != nil {
		return fmt.Errorf("parse environment variables: %w", err)
	}

	// sets up logger based on envconfig
	logger.Setup(cfg.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Infof("naisdevice API server %s starting up", version.Version)

	wireguardPrefix, err := netip.ParsePrefix(cfg.WireGuardNetworkAddress)
	if err != nil {
		return fmt.Errorf("parse wireguard network address: %w", err)
	}

	ipAllocator := database.NewIPAllocator(wireguardPrefix, []string{cfg.WireGuardIP})
	db, err := database.New(ctx, cfg.DBPath, ipAllocator, !cfg.KolideEventHandlerEnabled)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}

	log.Infof("Loading user sessions from database...")

	sessions := auth.NewSessionStore(db)
	err = sessions.Warmup(ctx)
	if err != nil {
		return fmt.Errorf("warm session cache from database: %w", err)
	}

	switch cfg.DeviceAuthenticationProvider {
	case "azure":
		log.Infof("Fetching Azure OIDC configuration...")
		err = cfg.Azure.SetupJwkSetAutoRefresh()
		if err != nil {
			return fmt.Errorf("fetch Azure jwks: %w", err)
		}

		authenticator = auth.NewAuthenticator(cfg.Azure, db, sessions)
		log.Infof("Azure OIDC authenticator configured to authenticate device sessions.")
	case "google":
		log.Infof("Setting up Google OIDC configuration...")
		err = cfg.Google.SetupJwkSetAutoRefresh()
		if err != nil {
			return fmt.Errorf("set up Google jwks: %w", err)
		}

		authenticator = auth.NewGoogleAuthenticator(cfg.Google, db, sessions)
		log.Infof("Google OIDC authenticator configured to authenticate device sessions.")
	default:
		authenticator = auth.NewMockAuthenticator(sessions)
		log.Warnf("Device authentication DISABLED! Do not run this configuration in production!")
		log.Warnf("To enable device authentication, specify auth provider with --device-authentication-provider=azure|google")
	}

	if cfg.WireGuardEnabled {
		log.Infof("Setting up WireGuard integration...")

		err = setupInterface(cfg.WireGuardIP, wireguardPrefix)
		if err != nil {
			return fmt.Errorf("set up WireGuard interface: %w", err)
		}

		key, err := wg.ReadOrCreatePrivateKey(cfg.WireGuardPrivateKeyPath, log.WithField("component", "wireguard"))
		if err != nil {
			return fmt.Errorf("generate WireGuard private key: %w", err)
		}
		cfg.WireGuardPrivateKey = key

		w := wireguard.New(cfg, db, cfg.WireGuardPrivateKey)

		go SyncLoop(w)

		log.Infof("WireGuard successfully configured.")

	} else {
		log.Warnf("WireGuard integration DISABLED! Do not run this configuration in production!")
	}

	deviceUpdates := make(chan *kolidepb.DeviceEvent, 64)

	if cfg.KolideEventHandlerEnabled {
		if len(cfg.KolideEventHandlerAddress) == 0 {
			return fmt.Errorf("--kolide-event-handler-address %w", errRequiredArgNotSet)
		}

		go func() {
			log.Infof("Kolide event handler stream starting on %s", cfg.KolideEventHandlerAddress)
			err := kolide.DeviceEventStreamer(ctx, cfg.KolideEventHandlerAddress, cfg.KolideEventHandlerToken, cfg.KolideEventHandlerSecure, deviceUpdates)
			if err != nil {
				log.Errorf("Kolide event streamer finished: %s", err)
			}
			cancel()
		}()
	}

	if len(cfg.BootstrapAPIURL) > 0 {
		parts := strings.Split(cfg.BootstrapApiCredentials, ":")
		username, password := parts[0], parts[1]

		en := enroller.Enroller{
			Client:             basicauth.Transport{Username: username, Password: password}.Client(),
			DB:                 db,
			BootstrapAPIURL:    cfg.BootstrapAPIURL,
			APIServerPublicKey: string(cfg.WireGuardPrivateKey.Public()),
			APIServerEndpoint:  cfg.Endpoint,
			APIServerIP:        cfg.WireGuardIP,
		}

		go en.WatchDeviceEnrollments(ctx)
	}

	if cfg.AutoEnrollEnabled {
		enrollPeers := append(cfg.StaticPeers(), cfg.APIServerPeer())
		e, err := enroller.NewAutoEnroll(ctx, db, enrollPeers, cfg.GRPCBindAddress, log.WithField("component", "auto-enroller"))
		if err != nil {
			return err
		}
		go func() {
			err := e.Run(ctx)
			if err != nil {
				log.WithError(err).Error("Run AutoEnroll failed")
				cancel()
			}
		}()
	}

	jitaClient := jita.New(cfg.JitaUsername, cfg.JitaPassword, cfg.JitaUrl)
	if cfg.JitaEnabled {
		go SyncJitaContinuosly(ctx, jitaClient)
	}

	switch cfg.GatewayConfigurer {
	case "bucket":
		buck := bucket.NewClient(cfg.GatewayConfigBucketName, cfg.GatewayConfigBucketObjectName)

		updater := gatewayconfigurer.GatewayConfigurer{
			DB:           db,
			Bucket:       buck,
			SyncInterval: gatewayConfigSyncInterval,
		}

		go updater.SyncContinuously(ctx)
	case "metadata":
		updater := gatewayconfigurer.NewGoogleMetadata(db, log.WithField("component", "gatewayconfigurer"))
		go updater.SyncContinuously(ctx, gatewayConfigSyncInterval)
	default:
		log.Warn("no valid gateway configurer set, gateways won't be updated.")
	}

	if cfg.ControlPlaneAuthenticationEnabled {
		apiKeys, err := config.Credentials(cfg.AdminCredentialEntries)
		if err != nil {
			return fmt.Errorf("parse admin credentials: %w", err)
		}

		if len(apiKeys) == 0 {
			return fmt.Errorf("control plane basic authentication enabled, but no admin credentials provided (try --admin-credential-entries)")
		}

		promauth, err := config.Credentials(cfg.PrometheusCredentialEntries)
		if err != nil {
			return fmt.Errorf("parse prometheus credentials: %w", err)
		}

		if len(promauth) == 0 {
			return fmt.Errorf("control plane basic authentication enabled, but no prometheus credentials provided (try --prometheus-credential-entries)")
		}

		adminAuthenticator = auth.NewAPIKeyAuthenticator(apiKeys)
		gatewayAuthenticator = auth.NewGatewayAuthenticator(db)
		prometheusAuthenticator = auth.NewAPIKeyAuthenticator(promauth)

		log.Infof("Control plane authentication enabled.")

	} else {
		adminAuthenticator = auth.NewMockAPIKeyAuthenticator()
		gatewayAuthenticator = auth.NewMockAPIKeyAuthenticator()
		prometheusAuthenticator = auth.NewMockAPIKeyAuthenticator()

		log.Warnf("Control plane authentication DISABLED! Do not run this configuration in production!")
	}

	grpcHandler := api.NewGRPCServer(
		ctx,
		db,
		authenticator,
		adminAuthenticator,
		gatewayAuthenticator,
		prometheusAuthenticator,
		jitaClient,
		sessions,
	)

	grpcServer := grpc.NewServer()
	pb.RegisterAPIServerServer(grpcServer, grpcHandler)

	grpcListener, err := net.Listen("tcp", cfg.GRPCBindAddress)
	if err != nil {
		return fmt.Errorf("unable to set up gRPC server: %w", err)
	}

	updateDevice := func(event *kolidepb.DeviceEvent) error {
		device, err := kolide.LookupDevice(ctx, db, event)
		if err != nil {
			return err
		}

		changed := false
		if device.Healthy != event.GetState().Healthy() {
			changed = true
		}

		device.Healthy = event.GetState().Healthy()
		device.LastUpdated = event.GetTimestamp()
		err = db.UpdateDevices(ctx, []*pb.Device{device})
		if err != nil {
			return err
		}
		if changed {
			sessions.UpdateDevice(device)
			grpcHandler.SendDeviceConfiguration(device)
			grpcHandler.SendAllGatewayConfigurations()
		}
		return nil
	}

	// TODO: remove when we've improved JITA
	// This triggers sync every 10 sec to let gateways know if someone has JITA'd
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				grpcHandler.SendAllGatewayConfigurations()

			case <-ctx.Done():
				log.Infof("Stopped gateway-sync hack")
				return
			}
		}
	}()

	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		err := apiserver_metrics.Serve(cfg.PrometheusAddr)
		if err != nil {
			log.Errorf("metrics server shut down with error; killing apiserver process: %s", err)
			cancel()
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

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigs
		log.Infof("Received signal %s", s)
		cancel()
	}()

	defer grpcServer.Stop()

	go func() {
		for event := range deviceUpdates {
			if err := updateDevice(event); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					log.Debugf("Update device health: %s", err)
				} else {
					log.Errorf("Update device health: %s", err)
				}
			}
		}
	}()

	<-ctx.Done()

	log.Warnf("Program context canceled; shutting down.")
	return nil
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

func setupInterface(ip string, prefix netip.Prefix) error {
	if err := exec.Command("ip", "link", "del", "wg0").Run(); err != nil {
		log.Infof("Pre-deleting WireGuard interface (ok if this fails): %v", err)
	}

	run := func(commands [][]string) error {
		for _, s := range commands {
			cmd := exec.Command(s[0], s[1:]...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("running %v: %w: %v", cmd, err, string(out))
			} else {
				log.Debugf("%v: %v\n", cmd, string(out))
			}
		}
		return nil
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", "wg0", "type", "wireguard"},
		{"ip", "link", "set", "wg0", "mtu", "1360"},
		{"ip", "address", "add", "dev", "wg0", fmt.Sprintf("%s/%d", ip, prefix.Bits())},
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

func SyncJitaContinuosly(ctx context.Context, j jita.Client) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			log.Debug("Updating jita privileged users")
			err := j.UpdatePrivilegedUsers()
			if err != nil {
				log.Errorf("Updating jita privileged users: %s", err)
			}

		case <-ctx.Done():
			log.Infof("Stopped jita-sync")
			return
		}
	}
}
