package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	gateway_agent "github.com/nais/device/internal/gateway-agent"
	"github.com/nais/device/internal/gateway-agent/config"
	"github.com/nais/device/internal/passwordhash"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/internal/program"
	"github.com/nais/device/internal/pubsubenroll"
	"github.com/nais/device/internal/wireguard"

	"github.com/nais/device/internal/logger"

	"github.com/coreos/go-iptables/iptables"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/nais/device/internal/version"
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
	log := logger.Setup(cfg.LogLevel).WithField("component", "main")
	err := run(log, cfg)
	if err != nil {
		log.WithError(err).Error("running gateway-agent")
	}
}

func run(log *logrus.Entry, cfg config.Config) error {
	ctx, cancel := program.MainContext(1 * time.Second)
	defer cancel()

	err := envconfig.Process("GATEWAY_AGENT", &cfg)
	if err != nil {
		return fmt.Errorf("read environment configuration: %w", err)
	}

	logger.Setup(cfg.LogLevel)

	log.WithFields(version.LogFields).Info("starting gateway-agent")

	staticPeers := cfg.StaticPeers()
	if cfg.AutoEnroll {
		log.Info("auto bootstrap enabled")
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
	go gateway_agent.Serve(log.WithField("component", "prometheus"), cfg.PrometheusAddr)

	var netConf wireguard.NetworkConfigurer
	if cfg.EnableRouting {
		err = cfg.ValidateWireGuard()
		if err != nil {
			return fmt.Errorf("cannot enable routing: %w", err)
		}
		iptablesV4, err := iptables.NewWithProtocol(iptables.ProtocolIPv4)
		_ = iptablesV4 // workaround as statickcheck thinks this in unused (but only on macos :shrug:)
		if err != nil {
			return fmt.Errorf("setup iptables: %w", err)
		}
		iptablesV6, err := iptables.NewWithProtocol(iptables.ProtocolIPv6)
		_ = iptablesV6 // workaround as statickcheck thinks this in unused (but only on macos :shrug:)
		if err != nil {
			return fmt.Errorf("setup iptables: %w", err)
		}
		router, err := NewRouter()
		if err != nil {
			return fmt.Errorf("setup routing: %w", err)
		}
		netConf, err = wireguard.NewConfigurer(
			log.WithField("component", "network-configurer"),
			cfg.WireGuardConfigPath, cfg.WireGuardIPv4, cfg.WireGuardIPv6, cfg.PrivateKey, wireguardInterface, wireguardListenPort, iptablesV4, iptablesV6, router)
		if err != nil {
			return fmt.Errorf("setup wireguard configurer: %w", err)
		}
	} else {
		netConf = wireguard.NewNoOpConfigurer(log.WithField("component", "network-configurer"))
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

	log.WithField("url", cfg.APIServerURL).Info("attempting gRPC connection to apiserver")
	apiserver, err := grpc.NewClient(
		cfg.APIServerURL,
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             2 * time.Second,
			PermitWithoutStream: false,
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("unable to connect to api server: %w", err)
	}

	defer apiserver.Close()

	apiserverClient := pb.NewAPIServerClient(apiserver)

	for attempt := 0; attempt < maxReconnectAttempts; attempt++ {
		err := gateway_agent.SyncFromStream(ctx, log, cfg.Name, cfg.APIServerPassword, staticPeers, apiserverClient, netConf)
		if err != nil {
			code := status.Code(err)
			if code == codes.Unauthenticated {
				// invalid auth is probably not going to fix itself,
				// so we terminate here to let the OS restart us.
				return err
			}
			log.WithError(err).WithField("attempt", attempt).Error("failed, retrying")
			log.WithField("backoff", grpcConnectBackoff).Debug("sleep before retry...")
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
