package main

import (
	"context"
	"fmt"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	g "github.com/nais/device/pkg/gateway-agent"
	"github.com/nais/device/pkg/passwordhash"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/pubsubenroll"
	"github.com/nais/device/pkg/wireguard"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nais/device/pkg/logger"

	"github.com/coreos/go-iptables/iptables"
	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var cfg = g.DefaultConfig()

const (
	grpcConnectBackoff   = 5 * time.Second
	wireguardInterface   = "wg0"
	wireguardListenPort  = 51820
	enrollTimeout        = 20 * time.Second
	maxReconnectAttempts = 10
)

func init() {
	flag.BoolVar(&cfg.EnableRouting, "enable-routing", cfg.EnableRouting, "enable-routing enables setting up interface and configuring of WireGuard")
	flag.StringVar(&cfg.APIServerEndpoint, "apiserver-endpoint", cfg.APIServerEndpoint, "WireGuard public endpoint at API server, host:port")
	flag.StringVar(&cfg.APIServerPassword, "apiserver-password", cfg.APIServerPassword, "password to access apiserver")
	flag.StringVar(&cfg.APIServerPublicKey, "apiserver-public-key", cfg.APIServerPublicKey, "API server's WireGuard public key")
	flag.StringVar(&cfg.APIServerPrivateIP, "apiserver-private-ip", cfg.APIServerPrivateIP, "API server's WireGuard IP address")
	flag.StringVar(&cfg.APIServerURL, "api-server-url", cfg.APIServerURL, "prometheus tunnel ip")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "gateway-agent config directory")
	flag.StringVar(&cfg.DeviceIP, "device-ip", cfg.DeviceIP, "IP address to use in WireGuard VPN")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "log level")
	flag.StringVar(&cfg.Name, "name", cfg.Name, "gateway name")
	flag.StringVar(&cfg.PrivateKey, "private-key", cfg.PrivateKey, "wireguard private key")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.PrometheusPublicKey, "prometheus-public-key", cfg.PrometheusPublicKey, "prometheus public key")
	flag.StringVar(&cfg.PrometheusTunnelIP, "prometheus-tunnel-ip", cfg.PrometheusTunnelIP, "prometheus tunnel ip")
	flag.BoolVar(&cfg.AutoBootstrap, "auto-bootstrap", cfg.AutoBootstrap, "Auto bootstrap using pub/sub. Uses Google ADC.")
}

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Fatalf("Running gateway-agent: %s", err)
	}
}

func run() error {
	var err error

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err = envconfig.Process("GATEWAY_AGENT", &cfg)
	if err != nil {
		return fmt.Errorf("read environment configuration: %w", err)
	}

	logger.Setup(cfg.LogLevel)

	log.Infof("gateway-agent version %s, revision: %s", version.Version, version.Revision)

	g.InitializeMetrics(cfg.Name, version.Version)
	go g.Serve(cfg.PrometheusAddr)

	log.Info("starting gateway-agent")

	staticPeers := cfg.StaticPeers()
	if cfg.AutoBootstrap {
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

		ecfg, err := pubsubenroll.New(
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

		cfg.PrivateKey = string(privateKey.Private())
		cfg.APIServerURL = enrollResp.APIServerGRPCAddress
		cfg.DeviceIP = enrollResp.WireGuardIP

		staticPeers = wireguard.MakePeers(nil, enrollResp.Peers)
	}

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
		netConf = wireguard.NewConfigurer(cfg.WireGuardConfigPath, cfg.DeviceIP, cfg.PrivateKey, wireguardInterface, wireguardListenPort, ipTables)
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
		grpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("unable to connect to api server: %w", err)
	}

	defer apiserver.Close()

	apiserverClient := pb.NewAPIServerClient(apiserver)

	for attempt := 0; attempt < maxReconnectAttempts; attempt++ {
		err := g.SyncFromStream(ctx, cfg, apiserverClient, netConf)
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
}
