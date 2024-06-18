package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/nais/device/internal/apiserver/api"
	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/bucket"
	"github.com/nais/device/internal/apiserver/config"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/enroller"
	"github.com/nais/device/internal/apiserver/gatewayconfigurer"
	"github.com/nais/device/internal/apiserver/jita"
	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/pb"
	"github.com/nais/device/internal/wireguard"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func setupGatewayConfigurer(ctx context.Context, cfg config.Config, db database.Database, log *logrus.Entry) {
	switch cfg.GatewayConfigurer {
	case "bucket":
		buck := bucket.NewClient(cfg.GatewayConfigBucketName, cfg.GatewayConfigBucketObjectName)
		updater := gatewayconfigurer.NewGatewayConfigurer(log.WithField("component", "gatewayconfigurer"), db, buck, gatewayConfigSyncInterval)
		go updater.SyncContinuously(ctx)
	case "metadata":
		updater := gatewayconfigurer.NewGoogleMetadata(db, log.WithField("component", "gatewayconfigurer"))
		go updater.SyncContinuously(ctx, gatewayConfigSyncInterval)
	default:
		log.Warn("no gateway configurer set, gateways won't be updated.")
	}
}

func setupKolideEventHandler(ctx context.Context, cfg config.Config, db database.Database, kolideClient kolide.Client, sessions auth.SessionStore, grpcHandler API, log *logrus.Entry) error {
	if cfg.KolideEventHandlerEnabled {
		if len(cfg.KolideEventHandlerAddress) == 0 {
			return fmt.Errorf("kolide-event-handler-address not configured")
		}
		onEvent := func(externalId string) {
			failures, err := kolideClient.GetDeviceFailures(ctx, externalId)
			if err != nil {
				log.WithError(err).Error("onEvent: get device failures")
				return
			}

			device, err := db.ReadDeviceByExternalID(ctx, externalId)
			if err != nil {
				log.WithError(err).Error("onEvent: read device by external ID")
				return
			}

			lastUpdated := time.Now()
			if err := db.UpdateSingleDevice(ctx, externalId, device.Serial, device.Platform, &lastUpdated, failures); err != nil {
				log.WithError(err).Error("onEvent: update device")
				return
			}

			device, err = db.ReadDeviceByExternalID(ctx, externalId)
			if err != nil {
				log.WithError(err).Error("onEvent: read device by external ID")
				return
			}

			sessions.UpdateDevice(device)
			grpcHandler.SendDeviceConfiguration(device)
			grpcHandler.SendAllGatewayConfigurations()
		}

		log.Infof("Kolide event handler stream starting on %v", cfg.KolideEventHandlerAddress)
		logger := log.WithField("component", "kolide-event-handler")

		go kolide.KolideEventHandler(ctx, db, cfg.KolideEventHandlerAddress, cfg.KolideEventHandlerToken, cfg.KolideEventHandlerSecure, kolideClient, onEvent, logger)
	}
	return nil
}

func setupAuthenticator(cfg config.Config, db database.Database, sessions auth.SessionStore, log logrus.FieldLogger) (auth.Authenticator, error) {
	switch cfg.DeviceAuthenticationProvider {
	case "azure":
		log.Infof("Fetching Azure OIDC configuration...")
		err := cfg.Azure.SetupJwkSetAutoRefresh()
		if err != nil {
			return nil, fmt.Errorf("fetch Azure jwks: %w", err)
		}

		log.Infof("Azure OIDC authenticator configured to authenticate device sessions.")
		return auth.NewAuthenticator(cfg.Azure, db, sessions), nil
	case "google":
		log.Infof("Setting up Google OIDC configuration...")
		err := cfg.Google.SetupJwkSetAutoRefresh()
		if err != nil {
			return nil, fmt.Errorf("set up Google jwks: %w", err)
		}

		log.Infof("Google OIDC authenticator configured to authenticate device sessions.")
		return auth.NewGoogleAuthenticator(cfg.Google, db, sessions), nil
	default:
		log.Warnf("Device authentication DISABLED! Do not run this configuration in production!")
		log.Warnf("To enable device authentication, specify auth provider with --device-authentication-provider=azure|google")
		return auth.NewMockAuthenticator(sessions), nil
	}
}

func setupControlplaneAuth(cfg config.Config, db database.Database, log logrus.FieldLogger) (auth.UsernamePasswordAuthenticator, auth.UsernamePasswordAuthenticator, auth.UsernamePasswordAuthenticator, error) {
	if cfg.ControlPlaneAuthenticationEnabled {
		apiKeys, err := config.Credentials(cfg.AdminCredentialEntries)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse admin credentials: %w", err)
		}

		if len(apiKeys) == 0 {
			return nil, nil, nil, fmt.Errorf("control plane basic authentication enabled, but no admin credentials provided (try --admin-credential-entries)")
		}

		promauth, err := config.Credentials(cfg.PrometheusCredentialEntries)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse prometheus credentials: %w", err)
		}

		if len(promauth) == 0 {
			return nil, nil, nil, fmt.Errorf("control plane basic authentication enabled, but no prometheus credentials provided (try --prometheus-credential-entries)")
		}

		adminAuth := auth.NewAPIKeyAuthenticator(apiKeys)
		gatewayAuth := auth.NewGatewayAuthenticator(db)
		prometheusAuth := auth.NewAPIKeyAuthenticator(promauth)

		log.Infof("Control plane authentication enabled.")
		return adminAuth, gatewayAuth, prometheusAuth, nil
	} else {
		adminAuth := auth.NewMockAPIKeyAuthenticator()
		gatewayAuth := auth.NewMockAPIKeyAuthenticator()
		prometheusAuth := auth.NewMockAPIKeyAuthenticator()

		log.Warnf("Control plane authentication DISABLED! Do not run this configuration in production!")

		return adminAuth, gatewayAuth, prometheusAuth, nil
	}
}

func setupKolideClient(ctx context.Context, cfg config.Config, db database.Database, log logrus.FieldLogger) (kolide.Client, error) {
	if !cfg.KolideIntegrationEnabled {
		return nil, nil
	}

	if cfg.KolideApiToken == "" {
		return nil, fmt.Errorf("kolide integration enabled but no kolide-api-token provided")
	}

	client := kolide.New(cfg.KolideApiToken, db, log.WithField("component", "kolide-client"))
	duration := time.Minute
	log.Infof("Kolide client configured, cache will auto refresh every %v", duration)
	go callEvery(ctx, client.RefreshCache, duration, log.WithField("component", "kolide-client"))
	return client, nil
}

func setupAutoEnroll(ctx context.Context, cfg config.Config, db database.Database, cancelMainContext context.CancelFunc, log logrus.FieldLogger) error {
	if cfg.AutoEnrollEnabled {
		enrollPeers := append(cfg.StaticPeers(), cfg.APIServerPeer())
		e, err := enroller.NewAutoEnroll(ctx, db, enrollPeers, cfg.GRPCBindAddress, log.WithField("component", "auto-enroller"))
		if err != nil {
			return err
		}
		go func() {
			if err := e.Run(ctx); err != nil {
				// if auto enroller stops working we need to exit
				log.WithError(err).Error("auto enroller stopped, cancelling program context")
				cancelMainContext()
			}
		}()
	}
	return nil
}

func setupJitaClient(ctx context.Context, cfg config.Config, log *logrus.Entry) jita.Client {
	if cfg.JitaEnabled {
		logger := log.WithField("component", "jita")
		jitaClient := jita.New(logger, cfg.JitaUsername, cfg.JitaPassword, cfg.JitaUrl)
		go callEvery(ctx, jitaClient.UpdatePrivilegedUsers, 10*time.Second, logger)
		return jitaClient
	}
	return nil
}

func syncFunc(db database.Database, netConf wireguard.NetworkConfigurer, staticPeers []*pb.Gateway) func(context.Context) error {
	return func(ctx context.Context) error {
		devices, err := db.ReadDevices(ctx)
		if err != nil {
			return fmt.Errorf("reading devices from database: %v", err)
		}

		gateways, err := db.ReadGateways(ctx)
		if err != nil {
			return fmt.Errorf("reading gateways from database: %v", err)
		}

		peers := wireguard.CastPeerList(staticPeers)
		peers = append(peers, wireguard.CastPeerList(devices)...)
		peers = append(peers, wireguard.CastPeerList(gateways)...)

		err = netConf.ApplyWireGuardConfig(peers)
		if err != nil {
			return fmt.Errorf("apply wireguard config: %v", err)
		}

		return nil
	}
}

func setupWireGuard(ctx context.Context, cfg config.Config, db database.Database, log *logrus.Entry) error {
	if cfg.WireGuardEnabled {
		log.Infof("Setting up WireGuard integration...")

		key, err := wireguard.ReadOrCreatePrivateKey(cfg.WireGuardPrivateKeyPath, log.WithField("component", "wireguard"))
		if err != nil {
			return fmt.Errorf("generate WireGuard private key: %w", err)
		}
		cfg.WireGuardPrivateKey = key

		netConf, err := wireguard.NewConfigurer(log.WithField("component", "network-configurer"), cfg.WireGuardConfigPath, cfg.WireGuardIPv4Prefix, cfg.WireGuardIPv6Prefix, string(cfg.WireGuardPrivateKey.Private()), "wg0", 51820, nil, nil, nil)
		if err != nil {
			return fmt.Errorf("create WireGuard configurer: %w", err)
		}

		err = netConf.SetupInterface()
		if err != nil {
			return fmt.Errorf("setup interface: %w", err)
		}

		sync := syncFunc(db, netConf, cfg.StaticPeers())
		go callEvery(ctx, sync, 20*time.Second, log.WithField("component", "wireguard"))

		log.Infof("WireGuard successfully configured.")
	} else {
		log.Warnf("WireGuard integration DISABLED! Do not run this configuration in production!")
	}
	return nil
}

type API interface {
	ReportOnlineGateways()
	SendDeviceConfiguration(*pb.Device)
	SendAllGatewayConfigurations()
}

func setupGRPCHandler(ctx context.Context, cfg config.Config, db database.Database, jitaClient jita.Client, sessions auth.SessionStore, kolideClient kolide.Client, exitProgram context.CancelFunc, log *logrus.Entry) (API, func(), error) {
	listener, err := net.Listen("tcp", cfg.GRPCBindAddress)
	if err != nil {
		return nil, func() {}, fmt.Errorf("unable to set up gRPC server: %w", err)
	}

	deviceAuth, err := setupAuthenticator(cfg, db, sessions, log)
	if err != nil {
		return nil, func() {}, err
	}

	adminAuth, gatewayAuth, prometheusAuth, err := setupControlplaneAuth(cfg, db, log)
	if err != nil {
		return nil, func() {}, err
	}
	grpcHandler := api.NewGRPCServer(
		ctx,
		log,
		db,
		deviceAuth,
		adminAuth,
		gatewayAuth,
		prometheusAuth,
		jitaClient,
		sessions,
		kolideClient,
	)

	grpcServer := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{MinTime: 9 * time.Second}),
		grpc.StatsHandler(otel.NewGRPCClientHandler(pb.APIServer_GetDeviceConfiguration_FullMethodName, pb.APIServer_GetGatewayConfiguration_FullMethodName)),
	)

	pb.RegisterAPIServerServer(grpcServer, grpcHandler)

	go func() {
		defer exitProgram()
		log.Infof("gRPC server starting on %v", cfg.GRPCBindAddress)
		if err := grpcServer.Serve(listener); err != nil {
			log.WithError(err).Error("gRPC server exited with error")
		}
	}()

	return grpcHandler, grpcServer.Stop, nil
}
