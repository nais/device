package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
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
	"github.com/nais/device/internal/apiserver/metrics"
	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/program"
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
		log.WithError(err).Error("parse configuration")
		os.Exit(1)
	}

	err = run(log, cfg)
	if err != nil {
		log.WithError(err).Error("unhandled error")
		os.Exit(1)
	} else {
		log.Info("naisdevice API server has shut down cleanly")
	}
}

func run(log *logrus.Entry, cfg config.Config) error {
	var authenticator auth.Authenticator
	var adminAuthenticator auth.UsernamePasswordAuthenticator
	var gatewayAuthenticator auth.UsernamePasswordAuthenticator
	var prometheusAuthenticator auth.UsernamePasswordAuthenticator

	ctx, cancel := program.MainContext(1 * time.Second)
	defer cancel()

	otelCancel, err := otel.SetupOTelSDK(ctx, "naisdevice-apiserver", log)
	if err != nil {
		return fmt.Errorf("setup OTel SDK: %s", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := otelCancel(ctx); err != nil {
			log.WithError(err).Error("shutdown OTel SDK")
		}
		cancel()
	}()

	log.WithFields(version.LogFields).Info("starting naisdevice apiserver")
	log.WithField("WireGuardIPv4Prefix", cfg.WireGuardIPv4Prefix).WithField("WireGuardIPv6Prefix", cfg.WireGuardIPv6Prefix).Info("networks config")

	wireguardPrefix, err := netip.ParsePrefix(cfg.WireGuardNetworkAddress)
	if err != nil {
		return fmt.Errorf("parse wireguard network address: %w", err)
	}

	v4Allocator := ip.NewV4Allocator(wireguardPrefix, []string{cfg.WireGuardIPv4Prefix.Addr().String()})
	v6Allocator := ip.NewV6Allocator(cfg.WireGuardIPv6Prefix)
	db, err := database.New(cfg.DBPath, v4Allocator, v6Allocator, !cfg.KolideEventHandlerEnabled)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}

	err = readd(ctx, db)
	if err != nil {
		log.WithError(err).Error("upsert IPv6")
	} else {
		log.Info("re-added all gateways and devices")
	}

	log.Info("loading user sessions from database...")

	sessions := auth.NewSessionStore(db)
	err = sessions.Warmup(ctx)
	if err != nil {
		return fmt.Errorf("warm session cache from database: %w", err)
	}

	switch cfg.DeviceAuthenticationProvider {
	case "azure":
		log.Info("fetching Azure OIDC configuration...")
		err = cfg.Azure.SetupJwkSetAutoRefresh(ctx)
		if err != nil {
			return fmt.Errorf("fetch Azure jwks: %w", err)
		}

		authenticator = auth.NewAuthenticator(cfg.Azure, db, sessions)
		log.Info("Azure OIDC authenticator configured to authenticate device sessions")
	case "google":
		log.Info("setting up Google OIDC configuration...")
		err = cfg.Google.SetupJwkSetAutoRefresh(ctx)
		if err != nil {
			return fmt.Errorf("set up Google jwks: %w", err)
		}

		authenticator = auth.NewGoogleAuthenticator(cfg.Google, db, sessions)
		log.Info("Google OIDC authenticator configured to authenticate device sessions")
	default:
		authenticator = auth.NewMockAuthenticator(sessions)
		log.Warn("device authentication DISABLED! Do not run this configuration in production!")
		log.Warn("to enable device authentication, specify auth provider with --device-authentication-provider=azure|google")
	}

	if cfg.WireGuardEnabled {
		log.Info("setting up WireGuard integration...")

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

		log.Info("WireGuard successfully configured")

	} else {
		log.Warn("WireGuard integration DISABLED! Do not run this configuration in production!")
	}

	deviceUpdates := make(chan *kolidepb.DeviceEvent, 64)

	if cfg.KolideEventHandlerEnabled {
		if len(cfg.KolideEventHandlerAddress) == 0 {
			return fmt.Errorf("kolide-event-handler-address not configured")
		}

		go func() {
			log.WithField("event_handler_address", cfg.KolideEventHandlerAddress).Info("Kolide event handler stream starting")
			err := kolide.DeviceEventStreamer(ctx,
				log.WithField("component", "kolide-event-handler"),
				cfg.KolideEventHandlerAddress,
				cfg.KolideEventHandlerToken,
				cfg.KolideEventHandlerSecure,
				deviceUpdates,
			)
			if err != nil {
				log.WithError(err).Error("Kolide event streamer finished")
			}
			cancel()
		}()
	}

	var kolideClient kolide.Client
	if cfg.KolideIntegrationEnabled {
		if cfg.KolideApiToken == "" {
			return fmt.Errorf("kolide integration enabled but no kolide-api-token provided")
		}

		kolideClient = kolide.New(cfg.KolideApiToken, db, log.WithField("component", "kolide-client"))

		go func() {
			log.Info("Kolide client configured, populating cache...")

			err := kolideClient.RefreshCache(ctx)
			if err != nil {
				log.WithError(err).Error("initial kolide cache warmup")
			}

			kolideRefreshInterval := 1 * time.Minute
			log.WithField("interval", kolideRefreshInterval).Info("Kolide cache populated, will auto refresh")
			sleep := time.NewTicker(kolideRefreshInterval)
			for {
				select {
				case <-ctx.Done():
					log.Info("kolide cache updater done")
					return
				case <-sleep.C:
					err := kolideClient.RefreshCache(ctx)
					if err != nil {
						log.WithError(err).Error("Kolide cache refresh")
					}
				}
			}
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
				log.WithError(err).Error("run AutoEnroll failed")
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
		log.Warn("no valid gateway configurer set, gateways won't be updated")
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

		log.Info("controlplane authentication enabled")
	} else {
		adminAuthenticator = auth.NewMockAPIKeyAuthenticator()
		gatewayAuthenticator = auth.NewMockAPIKeyAuthenticator()
		prometheusAuthenticator = auth.NewMockAPIKeyAuthenticator()

		log.Warn("controlplane authentication DISABLED! Do not run this configuration in production!")
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
		kolideClient,
	)

	opts := []grpc.ServerOption{
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{MinTime: 9 * time.Second}),
		grpc.StatsHandler(otel.NewGRPCClientHandler(pb.APIServer_GetDeviceConfiguration_FullMethodName, pb.APIServer_GetGatewayConfiguration_FullMethodName)),
	}

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterAPIServerServer(grpcServer, grpcHandler)

	grpcListener, err := net.Listen("tcp", cfg.GRPCBindAddress)
	if err != nil {
		return fmt.Errorf("unable to set up gRPC server: %w", err)
	}

	updateDevice := func(event *kolidepb.DeviceEvent) error {
		device, err := db.ReadDeviceByExternalID(ctx, event.GetExternalID())
		if err != nil {
			return fmt.Errorf("read device with external_id=%v: %w", event.GetExternalID(), err)
		}

		failures, err := kolideClient.GetDeviceFailures(ctx, device.ExternalID)
		if err != nil {
			return err
		}

		sessions.UpdateDevice(device)

		now := time.Now()
		err = db.UpdateSingleDevice(ctx, device.ExternalID, device.Serial, device.Platform, &now, failures)
		if err != nil {
			return err
		}

		grpcHandler.SendDeviceConfiguration(device)
		grpcHandler.SendAllGatewayConfigurations()
		return nil
	}

	go func() {
		log.WithField("address", cfg.PrometheusAddr).Info("serving metrics")
		err := metrics.Serve(cfg.PrometheusAddr)
		if err != nil {
			log.WithError(err).Error("metrics server shut down with error; killing apiserver process")
			cancel()
		}
	}()

	go func() {
		log.WithField("address", cfg.GRPCBindAddress).Info("gRPC server starting")
		err := grpcServer.Serve(grpcListener)
		if err != nil {
			log.WithError(err).Error("gRPC server exited with error")
		}
		cancel()
	}()

	untilContextDone := func(ctx context.Context, interval time.Duration, f func(context.Context) error) {
		ticker := time.NewTicker(interval)
		for {
			if err := f(ctx); err != nil {
				log.WithError(err).Error("run until program done wrapper")
			}

			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}

	// sync all devices continuously
	go untilContextDone(ctx, 1*time.Minute, grpcHandler.UpdateAllDevices)

	// initialize gateway metrics
	gateways, err := db.ReadGateways(ctx)
	if err != nil {
		return err
	}
	for _, gateway := range gateways {
		metrics.SetGatewayConnected(gateway.Name, false)
	}

	defer grpcServer.Stop()

	go func() {
		for event := range deviceUpdates {
			if err := updateDevice(event); err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					log.WithError(err).Error("update device health")
				}
			}
		}
	}()

	<-ctx.Done()

	log.Warn("program context canceled; shutting down")
	return nil
}

func readd(ctx context.Context, db database.Database) error {
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

func SyncLoop(ctx context.Context, log *logrus.Entry, db database.Database, netConf wg.NetworkConfigurer, staticPeers []*pb.Gateway) {
	log.Debug("starting config sync")

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
			log.WithError(err).Error("syncing wg config")
		}
	}
}

func SyncJitaContinuosly(ctx context.Context, log *logrus.Entry, j jita.Client) {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			log.Debug("updating jita privileged users")
			err := j.UpdatePrivilegedUsers(ctx)
			if err != nil {
				log.WithError(err).Error("updating jita privileged users")
			}

		case <-ctx.Done():
			log.Info("stopped jita-sync")
			return
		}
	}
}
