package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	g "github.com/nais/device/pkg/gateway-agent"
	"github.com/nais/device/pkg/pb"
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

var (
	cfg = g.DefaultConfig()
)

const (
	grpcConnectBackoff = 5 * time.Second
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

	flag.Parse()
}

func main() {
	err := run()
	if err != nil {
		log.Errorf("fatal: %s", err)
		os.Exit(1)
	}
}

func run() error {
	var err error

	err = envconfig.Process("GATEWAY_AGENT", &cfg)
	if err != nil {
		return fmt.Errorf("read environment configuration: %w", err)
	}

	logger.Setup(cfg.LogLevel)

	log.Infof("gateway-agent version %s, revision: %s", version.Version, version.Revision)

	g.InitializeMetrics(cfg.Name, version.Version)
	go g.Serve(cfg.PrometheusAddr)

	log.Info("starting gateway-agent")

	var netConf g.NetworkConfigurer
	if cfg.EnableRouting {
		err = cfg.ValidateWireGuard()
		if err != nil {
			return fmt.Errorf("cannot enable routing: %w", err)
		}
		ipTables, err := iptables.New()
		if err != nil {
			return fmt.Errorf("setup iptables: %w", err)
		}
		netConf = g.NewConfigurer(cfg, ipTables)
	} else {
		netConf = g.NewNoOpConfigurer()
	}

	err = netConf.SetupInterface()
	if err != nil {
		return fmt.Errorf("setup interface: %w", err)
	}

	err = netConf.SetupIPTables()
	if err != nil {
		return fmt.Errorf("setup iptables defaults: %w", err)
	}

	err = netConf.ApplyWireGuardConfig(nil)
	if err != nil {
		return fmt.Errorf("apply wireguard config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		log.Infof("Received signal %s", sig)
		cancel()
	}()

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

	for ctx.Err() == nil {
		err := g.SyncFromStream(ctx, cfg, apiserverClient, netConf)
		if err != nil {
			code := status.Code(err)
			if code == codes.Unauthenticated {
				// invalid auth is probably not going to fix itself,
				// so we terminate here to let the OS restart us.
				return err
			}
			log.Error(err)
			log.Debugf("Waiting %s before next retry...", grpcConnectBackoff)
			select {
			case <-ctx.Done(): //context cancelled
			case <-time.After(grpcConnectBackoff): //timeout
			}
		}
	}

	return nil
}
