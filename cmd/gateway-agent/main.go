package main

import (
	"context"
	"net/http"
	"path"
	"path/filepath"
	"time"

	"github.com/nais/device/pkg/basicauth"
	g "github.com/nais/device/pkg/gateway-agent"
	"github.com/nais/device/pkg/pb"

	"github.com/nais/device/pkg/logger"

	"github.com/coreos/go-iptables/iptables"
	"github.com/nais/device/pkg/version"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	cfg = g.DefaultConfig()
)

func init() {
	flag.StringVar(&cfg.Name, "name", cfg.Name, "gateway name")
	flag.StringVar(&cfg.ConfigDir, "config-dir", cfg.ConfigDir, "gateway-agent config directory")
	flag.StringVar(&cfg.PublicIP, "public-ip", cfg.PublicIP, "public gateway ip")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.PrometheusPublicKey, "prometheus-public-key", cfg.PrometheusPublicKey, "prometheus public key")
	flag.StringVar(&cfg.PrometheusTunnelIP, "prometheus-tunnel-ip", cfg.PrometheusTunnelIP, "prometheus tunnel ip")
	flag.BoolVar(&cfg.EnableRouting, "enable-routing", cfg.EnableRouting, "enable-routing enables setting up interface and configuring of WireGuard")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "log level")
	flag.StringVar(&cfg.EnrollmentToken, "enrollment-token", "is not set", "bootstrap-api enrollment token")

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
	if err := cfg.InitLocalConfig(); err != nil {
		log.Fatalf("Initializing local configuration: %v", err)
	}

	bootstrapper := g.Bootstrapper{
		Config:     &cfg,
		HTTPClient: &http.Client{Transport: &basicauth.Transport{Username: cfg.Name, Password: cfg.EnrollmentToken}},
	}

	var err error
	cfg.BootstrapConfig, err = bootstrapper.EnsureBootstrapConfig()

	if err != nil {
		log.Fatalf("Ensuring gateway is bootstrapped: %v", err)
	}

	log.Info("starting gateway-agent")

	var netConf g.NetworkConfigurer
	if cfg.EnableRouting {
		ipTables, err := iptables.New()
		if err != nil {
			log.Fatalf("setting up iptables %v", err)
		}
		netConf = g.NewConfigurer(cfg, ipTables)
	} else {
		netConf = &g.MockNetworkConfigurer{}
	}

	err = netConf.SetupInterface()
	if err != nil {
		log.Fatalf("setting up interface: %v", err)
	}

	err = netConf.SetupIPTables()
	if err != nil {
		log.Fatalf("setting up iptables defaults: %v", err)
	}

	err = netConf.ActuateWireGuardConfig(make([]*pb.Device, 0))
	if err != nil {
		log.Fatalf("actuating base config: %v", err)
	}

	for {
		ctx, cancel := context.WithCancel(context.Background())
		stream, err := g.GetGatewayConfig(ctx, cfg)
		if err != nil {
			log.Errorf("connecting to gateway config stream: %v", err)
			time.Sleep(1 * time.Second)
			cancel()
			continue
		}

		for {
			gwConfig, err := stream.Recv()
			if err != nil {
				log.Errorf("get gateway config: %v", err)
				cancel()
				break
			}
			g.ApplyGatewayConfig(netConf, gwConfig)
		}
	}
}
