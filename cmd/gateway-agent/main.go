package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nais/device/pkg/basicauth"
	g "github.com/nais/device/pkg/gateway-agent"
	"github.com/nais/device/pkg/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nais/device/pkg/logger"

	"github.com/coreos/go-iptables/iptables"
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
	flag.StringVar(&cfg.Name, "name", cfg.Name, "gateway name")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "gateway-agent config directory")
	flag.StringVar(&cfg.PublicIP, "public-ip", cfg.PublicIP, "public gateway ip")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.PrometheusPublicKey, "prometheus-public-key", cfg.PrometheusPublicKey, "prometheus public key")
	flag.StringVar(&cfg.PrometheusTunnelIP, "prometheus-tunnel-ip", cfg.PrometheusTunnelIP, "prometheus tunnel ip")
	flag.StringVar(&cfg.APIServerURL, "api-server-url", cfg.APIServerURL, "prometheus tunnel ip")
	flag.BoolVar(&cfg.EnableRouting, "enable-routing", cfg.EnableRouting, "enable-routing enables setting up interface and configuring of WireGuard")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "log level")
	flag.StringVar(&cfg.EnrollmentToken, "enrollment-token", cfg.EnrollmentToken, "bootstrap-api enrollment token")

	flag.Parse()

	logger.Setup(cfg.LogLevel)
	cfg.WireGuardConfigPath = path.Join(cfg.ConfigDir, "wg0.conf")
	cfg.PrivateKeyPath = path.Join(cfg.ConfigDir, "private.key")
	cfg.APIServerPasswordPath = path.Join(cfg.ConfigDir, "apiserver_password")
	cfg.BootstrapConfigPath = filepath.Join(cfg.ConfigDir, "bootstrapconfig.json")

	log.Infof("Version: %s, Revision: %s", version.Version, version.Revision)

	g.InitializeMetrics(cfg.Name, version.Version)
	go g.Serve(cfg.PrometheusAddr)
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

	err = cfg.InitLocalConfig()
	if err != nil {
		return fmt.Errorf("initialize local configuration: %v", err)
	}

	log.Info("starting gateway-agent")

	var netConf g.NetworkConfigurer
	if cfg.EnableRouting {
		bootstrapper := g.Bootstrapper{
			Config: &cfg,
			HTTPClient: &http.Client{
				Transport: &basicauth.Transport{
					Username: cfg.Name,
					Password: cfg.EnrollmentToken,
				},
			},
		}

		cfg.BootstrapConfig, err = bootstrapper.EnsureBootstrapConfig()
		if err != nil {
			return fmt.Errorf("ensure gateway is bootstrapped: %w", err)
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

	err = netConf.ApplyWireGuardConfig(make([]*pb.Device, 0))
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

	for ctx.Err() == nil {
		err := g.SyncFromStream(ctx, cfg, netConf)
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
