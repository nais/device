package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/internal/apiserver/api"
	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/bucket"
	"github.com/nais/device/internal/apiserver/config"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/enroller"
	"github.com/nais/device/internal/apiserver/gatewayconfigurer"
	"github.com/nais/device/internal/apiserver/ip"
	"github.com/nais/device/internal/apiserver/jita"
	"github.com/nais/device/internal/apiserver/kolide"
	apiserver_metrics "github.com/nais/device/internal/apiserver/metrics"
	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/version"
	wg "github.com/nais/device/internal/wireguard"
	kolidepb "github.com/nais/kolide-event-handler/pkg/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

const (
	gatewayConfigSyncInterval = 1 * time.Minute
	WireGuardSyncInterval     = 20 * time.Second
)

func main() {
	cfg := config.DefaultConfig()

	err := envconfig.Process("APISERVER", &cfg)
	if err != nil {
		fmt.Println("unable to process environment variables: %w", err)
		os.Exit(1)
	}

	// sets up default logger
	log := logger.Setup(cfg.LogLevel).WithField("component", "main")

	err = cfg.Parse() // sets dynamic defaults for some config values
	if err != nil {
		log.Errorf("parse configuration: %v", err)
		os.Exit(1)
	}

	err = run(log, cfg)
	if err != nil {
		log.Errorf("unhandled error: %s", err)
		os.Exit(1)
	} else {
		log.Info("naisdevice API server has shut down cleanly.")
	}
}

func run(log *logrus.Entry, cfg config.Config) error {
	var authenticator auth.Authenticator
	var adminAuthenticator auth.UsernamePasswordAuthenticator
	var gatewayAuthenticator auth.UsernamePasswordAuthenticator
	var prometheusAuthenticator auth.UsernamePasswordAuthenticator

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Infof("naisdevice API server %s starting up", version.Version)
	log.Infof("WireGuard IPv4 address: %v", cfg.WireGuardIPv4Prefix)
	log.Infof("WireGuard IPv6 address: %v", cfg.WireGuardIPv6Prefix)

	wireguardPrefix, err := netip.ParsePrefix(cfg.WireGuardNetworkAddress)
	if err != nil {
		return fmt.Errorf("parse wireguard network address: %w", err)
	}

	v4Allocator := ip.NewV4Allocator(wireguardPrefix, []string{cfg.WireGuardIPv4Prefix.Addr().String()})
	v6Allocator := ip.NewV6Allocator(cfg.WireGuardIPv6Prefix)
	db, err := database.New(ctx, cfg.DBPath, v4Allocator, v6Allocator, !cfg.KolideEventHandlerEnabled)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}

	err = readd(ctx, db)
	if err != nil {
		log.Errorf("upsert IPv6: %v", err)
	} else {
		log.Info("re-added all gateways and devices")
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

		key, err := wg.ReadOrCreatePrivateKey(cfg.WireGuardPrivateKeyPath, log.WithField("component", "wireguard"))
		if err != nil {
			return fmt.Errorf("generate WireGuard private key: %w", err)
		}
		cfg.WireGuardPrivateKey = key

		netConf, err := wg.NewConfigurer(log.WithField("component", "network-configurer"), cfg.WireGuardConfigPath, cfg.WireGuardIPv4Prefix, cfg.WireGuardIPv6Prefix, string(cfg.WireGuardPrivateKey.Private()), "wg0", 51820, nil, nil, nil)
		if err != nil {
			return fmt.Errorf("create WireGuard configurer: %w", err)
		}

		err = netConf.SetupInterface()
		if err != nil {
			return fmt.Errorf("setup interface: %w", err)
		}

		go SyncLoop(ctx, log, db, netConf, cfg.StaticPeers())

		log.Infof("WireGuard successfully configured.")

	} else {
		log.Warnf("WireGuard integration DISABLED! Do not run this configuration in production!")
	}

	deviceUpdates := make(chan *kolidepb.DeviceEvent, 64)

	if cfg.KolideEventHandlerEnabled {
		if len(cfg.KolideEventHandlerAddress) == 0 {
			return fmt.Errorf("kolide-event-handler-address not configured")
		}

		go func() {
			log.Infof("Kolide event handler stream starting on %s", cfg.KolideEventHandlerAddress)
			err := kolide.DeviceEventStreamer(ctx,
				log.WithField("component", "kolide-event-handler"),
				cfg.KolideEventHandlerAddress,
				cfg.KolideEventHandlerToken,
				cfg.KolideEventHandlerSecure,
				deviceUpdates,
			)
			if err != nil {
				log.Errorf("Kolide event streamer finished: %s", err)
			}
			cancel()
		}()
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

	jitaClient := jita.New(log.WithField("component", "jita"), cfg.JitaUsername, cfg.JitaPassword, cfg.JitaUrl)
	if cfg.JitaEnabled {
		go SyncJitaContinuosly(ctx, log, jitaClient)
	}

	switch cfg.GatewayConfigurer {
	case "bucket":
		buck := bucket.NewClient(cfg.GatewayConfigBucketName, cfg.GatewayConfigBucketObjectName)
		updater := gatewayconfigurer.NewGatewayConfigurer(log.WithField("component", "gatewayconfigurer"), db, buck, gatewayConfigSyncInterval)
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
		log,
		db,
		authenticator,
		adminAuthenticator,
		gatewayAuthenticator,
		prometheusAuthenticator,
		jitaClient,
		sessions,
	)

	opts := []grpc.ServerOption{
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{MinTime: 9 * time.Second}),
	}

	grpcServer := grpc.NewServer(opts...)
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

func readd(ctx context.Context, db database.APIServer) error {
	gateways, err := db.ReadGateways(ctx)
	if err != nil {
		return err
	}
	for _, gateway := range gateways {
		if gateway.Ipv6 != "" {
			continue
		}
		if err := db.AddGateway(ctx, gateway); err != nil {
			return err
		}
	}

	devices, err := db.ReadDevices(ctx)
	if err != nil {
		return err
	}
	for _, device := range devices {
		if device.Ipv6 != "" {
			continue
		}
		if err := db.AddDevice(ctx, device); err != nil {
			return err
		}
	}

	return nil
}

func SyncLoop(ctx context.Context, log *logrus.Entry, db database.APIServer, netConf wg.NetworkConfigurer, staticPeers []*pb.Gateway) {
	log.Debugf("Starting config sync")

	sync := func(ctx context.Context) error {
		devices, err := db.ReadDevices(ctx)
		if err != nil {
			return fmt.Errorf("reading devices from database: %v", err)
		}

		gateways, err := db.ReadGateways(ctx)
		if err != nil {
			return fmt.Errorf("reading gateways from database: %v", err)
		}

		peers := wg.CastPeerList(staticPeers)
		peers = append(peers, wg.CastPeerList(devices)...)
		peers = append(peers, wg.CastPeerList(gateways)...)

		err = netConf.ApplyWireGuardConfig(peers)
		if err != nil {
			return fmt.Errorf("apply wireguard config: %v", err)
		}

		return nil
	}

	ticker := time.NewTicker(WireGuardSyncInterval)
	for range ticker.C {
		ctx, cancel := context.WithTimeout(ctx, WireGuardSyncInterval)
		err := sync(ctx)
		cancel()
		if err != nil {
			log.Errorf("syncing wg config: %s", err)
		}
	}
}

func SyncJitaContinuosly(ctx context.Context, log *logrus.Entry, j jita.Client) {
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
