package main

import (
	g "github.com/nais/device/gateway-agent"
	"github.com/nais/device/pkg/secretmanager"
	"path"
	"path/filepath"
	"time"

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
	flag.BoolVar(&cfg.DevMode, "development-mode", cfg.DevMode, "development mode avoids setting up interface and configuring WireGuard")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "log level")

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

	//TODO inputvar
	secretManager, err := secretmanager.New("nais-device")
	if err != nil {
		log.Fatalf("Initializing secret manager: %v", err)
	}

	bootstrapper := g.Bootstrapper{
		SecretManager: secretManager,
		Config:        &cfg,
	}

	cfg.BootstrapConfig, err = bootstrapper.GetBootstrapConfig()

	if err != nil {
		log.Fatalf("Ensuring gateway is bootstrapped: %v", err)
	}

	log.Info("starting gateway-agent")

	if !cfg.DevMode {
		if err := g.SetupInterface(cfg.BootstrapConfig.DeviceIP); err != nil {
			log.Fatalf("setting up interface: %v", err)
		}
		var err error
		cfg.IPTables, err = iptables.New()
		if err != nil {
			log.Fatalf("setting up iptables %v", err)
		}

		err = g.SetupIptables(cfg)
		if err != nil {
			log.Fatalf("Setting up iptables defaults: %v", err)
		}
	} else {
		log.Infof("Skipping interface setup")
	}

	baseConfig := g.GenerateBaseConfig(cfg)

	if err := g.ActuateWireGuardConfig(baseConfig, cfg.WireGuardConfigPath); err != nil && !cfg.DevMode {
		log.Fatalf("actuating base config: %v", err)
	}

	for range time.NewTicker(10 * time.Second).C {
		log.Infof("getting config")
		gatewayConfig, err := g.GetGatewayConfig(cfg)
		if err != nil {
			log.Error(err)
			g.FailedConfigFetches.Inc()
			continue
		}

		g.LastSuccessfulConfigFetch.SetToCurrentTime()

		log.Debugf("%+v\n", gatewayConfig)

		// skip side-effects for local development
		if cfg.DevMode {
			continue
		}

		if c, err := g.ConnectedDeviceCount(); err != nil {
			log.Errorf("Getting connected device count: %v", err)
		} else {
			g.ConnectedDevices.Set(float64(c))
		}

		peerConfig := g.GenerateWireGuardPeers(gatewayConfig.Devices)
		if err := g.ActuateWireGuardConfig(baseConfig+peerConfig, cfg.WireGuardConfigPath); err != nil {
			log.Errorf("actuating WireGuard config: %v", err)
		}

		err = g.ForwardRoutes(cfg, gatewayConfig.Routes)
		if err != nil {
			log.Errorf("forwarding routes: %v", err)
		}
	}
}
