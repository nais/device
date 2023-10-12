package main

import (
	"context"
	"fmt"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/nais/device/pkg/gateway-agent"
	"github.com/nais/device/pkg/gateway-agent/config"
	"github.com/nais/device/pkg/passwordhash"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/pubsubenroll"
	"github.com/nais/device/pkg/wireguard"

	"github.com/nais/device/pkg/logger"

	"github.com/coreos/go-iptables/iptables"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/version"
)

const (
	grpcConnectBackoff   = 5 * time.Second
	wireguardInterface   = "wg0"
	wireguardListenPort  = 51820
	enrollTimeout        = 20 * time.Second
	maxReconnectAttempts = 24 // ~ 2 minutes
)

func main() {
	cfg := config.DefaultConfig()
	err := run(cfg)
	if err != nil {
		log.Fatalf("Running gateway-agent: %s", err)
	}
}

func run(cfg config.Config) error {
	var err error

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err = envconfig.Process("GATEWAY_AGENT", &cfg)
	if err != nil {
		return fmt.Errorf("read environment configuration: %w", err)
	}

	logger.Setup(cfg.LogLevel)

	log.Infof("gateway-agent version %s, revision: %s", version.Version, version.Revision)

	log.Info("starting gateway-agent")

	staticPeers := cfg.StaticPeers()
	if cfg.AutoEnroll {
		log.Info("Auto bootstrap enabled")
		password, hashedPassword, err := passwordhash.GeneratePasswordAndHash()
		if err != nil {
			return err
		}
		cfg.APIServerPassword = password

		privateKey, err := wireguard.ReadOrCreatePrivateKey(
			filepath.Join(cfg.ConfigDir, "private.key"),
			log.WithField("component", "wireguard"),
		)
		if err != nil {
			return fmt.Errorf("get private key: %w", err)
		}

		ecfg, err := pubsubenroll.NewGatewayClient(
			ctx,
			privateKey.Public(),
			hashedPassword,
			wireguardListenPort,
			log.WithField("component", "bootstrap"),
		)
		if err != nil {
			return fmt.Errorf("create enroll config: %w", err)
		}

		ctx, cancel := context.WithTimeout(ctx, enrollTimeout)
		enrollResp, err := ecfg.Bootstrap(ctx)
		cancel()
		if err != nil {
			return fmt.Errorf("auto-bootstrap: %w", err)
		}

		cfg.Name = ecfg.Name
		cfg.PrivateKey = string(privateKey.Private())
		cfg.APIServerURL = enrollResp.APIServerGRPCAddress
		cfg.DeviceIPv4 = enrollResp.WireGuardIPv4

		staticPeers = wireguard.CastPeerList(enrollResp.Peers)
	}

	err = cfg.Parse()
	if err != nil {
		return fmt.Errorf("parse configuration: %w", err)
	}

	gateway_agent.InitializeMetrics(cfg.Name, version.Version)
	go gateway_agent.Serve(cfg.PrometheusAddr)

	var netConf wireguard.NetworkConfigurer
	if cfg.EnableRouting {
		err = cfg.ValidateWireGuard()
		if err != nil {
			return fmt.Errorf("cannot enable routing: %w", err)
		}
		ipTables, err := iptables.New()
		if err != nil {
			return fmt.Errorf("setup iptables: %w", err)
		}
		netConf = wireguard.NewConfigurer(cfg.WireGuardConfigPath, cfg.WireGuardIPv4, cfg.WireGuardIPv6, cfg.PrivateKey, wireguardInterface, wireguardListenPort, ipTables)
	} else {
		netConf = wireguard.NewNoOpConfigurer()
	}

	err = netConf.SetupInterface()
	if err != nil {
		return fmt.Errorf("setup interface: %w", err)
	}

	err = netConf.SetupIPTables()
	if err != nil {
		return fmt.Errorf("setup iptables defaults: %w", err)
	}

	err = netConf.ApplyWireGuardConfig(staticPeers)
	if err != nil {
		return fmt.Errorf("apply wireguard config: %w", err)
	}

	log.Infof("Attempting gRPC connection to API server on %s...", cfg.APIServerURL)
	apiserver, err := grpc.DialContext(
		ctx,
		cfg.APIServerURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("unable to connect to api server: %w", err)
	}

	defer apiserver.Close()

	apiserverClient := pb.NewAPIServerClient(apiserver)

	for attempt := 0; attempt < maxReconnectAttempts; attempt++ {
		err := gateway_agent.SyncFromStream(ctx, cfg.Name, cfg.APIServerPassword, staticPeers, apiserverClient, netConf)
		if err != nil {
			code := status.Code(err)
			if code == codes.Unauthenticated {
				// invalid auth is probably not going to fix itself,
				// so we terminate here to let the OS restart us.
				return err
			}
			log.Errorf("attempt %v: %v", attempt, err)
			log.Debugf("Waiting %s before next retry...", grpcConnectBackoff)
			select {
			case <-ctx.Done(): // context cancelled
				log.Info("context done, shutting down")
				return nil
			case <-time.After(grpcConnectBackoff): // timeout
			}
		}
	}
	log.Info("max reconnects reached, shutting down")
	return nil
}
