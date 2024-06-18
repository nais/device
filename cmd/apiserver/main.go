package main

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/config"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/ip"
	"github.com/nais/device/internal/apiserver/metrics"
	"github.com/nais/device/internal/logger"
	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/version"
	"github.com/sirupsen/logrus"
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
		log.Info("naisdevice API server has shut down cleanly.")
	}
}

func run(log *logrus.Entry, cfg config.Config) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	otelCancel, err := otel.SetupOTelSDK(ctx, "naisdevice-apiserver", log)
	if err != nil {
		return fmt.Errorf("setup OTel SDK: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := otelCancel(ctx); err != nil {
			log.WithError(err).Error("shutdown OTel SDK")
		}
		cancel()
	}()

	log.Infof("naisdevice API server %v starting up", version.Version)
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

	err = reAddWithIPv6Addressesb(ctx, db)
	if err != nil {
		log.WithError(err).Error("upsert IPv6")
	} else {
		log.Info("re-added all gateways and devices with IPv6 addresses")
	}

	log.Infof("Loading user sessions from database...")

	sessions := auth.NewSessionStore(db)
	err = sessions.Warmup(ctx)
	if err != nil {
		return fmt.Errorf("warm session cache from database: %w", err)
	}

	if err := setupWireGuard(ctx, cfg, db, log); err != nil {
		return err
	}

	kolideClient, err := setupKolideClient(ctx, cfg, db, log.WithField("component", "kolide-client"))
	if err != nil {
		return err
	}

	if err := setupAutoEnroll(ctx, cfg, db, cancel, log); err != nil {
		return err
	}

	jitaClient := setupJitaClient(ctx, cfg, log)

	setupGatewayConfigurer(ctx, cfg, db, log)

	grpcHandler, stopGRPCServer, err := setupGRPCHandler(ctx, cfg, db, jitaClient, sessions, kolideClient, cancel, log)
	if err != nil {
		return err
	}
	defer stopGRPCServer()

	// update connected gateways metric every 5 seconds
	go callEvery(ctx, func(_ context.Context) error { grpcHandler.ReportOnlineGateways(); return nil }, 5*time.Second, log.WithField("component", "gateway-metrics"))

	if err := setupKolideEventHandler(ctx, cfg, db, kolideClient, sessions, grpcHandler, log); err != nil {
		return err
	}

	// TODO: remove when we've improved JITA
	go callEvery(ctx, func(_ context.Context) error { grpcHandler.SendAllGatewayConfigurations(); return nil }, 10*time.Second, log.WithField("component", "gateway-sync"))

	go func() {
		defer cancel()
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		if err := metrics.Serve(cfg.PrometheusAddr); err != nil {
			log.WithError(err).Error("metrics server shut down with error")
		}
	}()

	<-ctx.Done()

	log.Warnf("Program context canceled; shutting down.")
	return nil
}

func reAddWithIPv6Addressesb(ctx context.Context, db database.Database) error {
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

func callEvery(ctx context.Context, f func(context.Context) error, duration time.Duration, log logrus.FieldLogger) {
	sleep := time.NewTicker(duration)
	for {
		err := f(ctx)
		if err != nil {
			log.WithError(err).Error("callEvery")
		}
		select {
		case <-ctx.Done():
			return
		case <-sleep.C:
		}
	}
}
