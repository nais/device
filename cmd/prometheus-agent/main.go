package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/pkg/pb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"google.golang.org/grpc"

	"github.com/nais/device/pkg/logger"
	prometheusagent "github.com/nais/device/pkg/prometheus-agent"
	"github.com/nais/device/pkg/version"
	"github.com/nais/device/pkg/wireguard"
)

var (
	cfg                        = prometheusagent.DefaultConfig()
	lastSuccessfulConfigUpdate = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "last_successful_config_update",
		Help:      "time since last successful prometheus config update",
		Namespace: "naisdevice",
		Subsystem: "prometheus_agent",
	})
)

const (
	wireguardInterface  = "wg0"
	wireguardListenPort = 51820
)

func init() {
	logger.Setup(cfg.LogLevel)
	flag.StringVar(&cfg.TunnelIP, "tunnel-ip", cfg.TunnelIP, "prometheus tunnel ip")
	flag.StringVar(&cfg.PrometheusAddr, "prometheus-address", cfg.PrometheusAddr, "prometheus listen address")
	flag.StringVar(&cfg.APIServerURL, "api-server-url", cfg.APIServerURL, "api server URL")
	flag.StringVar(&cfg.APIServerPublicKey, "api-server-public-key", cfg.APIServerPublicKey, "api server public key")
	flag.StringVar(&cfg.APIServerEndpoint, "api-server-endpoint", cfg.APIServerEndpoint, "api server WireGuard endpoint")
	flag.StringVar(&cfg.APIServerUsername, "api-server-username", cfg.APIServerUsername, "apiserver username")
	flag.StringVar(&cfg.APIServerPassword, "api-server-password", cfg.APIServerPassword, "apiserver password")
	flag.BoolVar(&cfg.WireGuardEnabled, "wireguard-enabled", cfg.WireGuardEnabled, "apiserver password")

	flag.Parse()
}

func main() {
	err := run()
	if err != nil {
		log.Fatalf("Running prometheus-agent: %s", err)
	}
}

func run() error {
	var err error
	ctx := context.Background()

	err = envconfig.Process("PROMETHEUS_AGENT", &cfg)
	if err != nil {
		return fmt.Errorf("read environment configuration: %w", err)
	}

	logger.Setup(cfg.LogLevel)

	log.Infof("gateway-agent version %s, revision: %s", version.Version, version.Revision)

	prometheus.MustRegister(lastSuccessfulConfigUpdate)

	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddr)
		_ = http.ListenAndServe(cfg.PrometheusAddr, promhttp.Handler())
	}()

	var netConf wireguard.NetworkConfigurer
	if cfg.WireGuardEnabled {
		err = cfg.ValidateWireGuard()
		if err != nil {
			return fmt.Errorf("cannot enable WireGuard: %w", err)
		}

		netConf = wireguard.NewConfigurer(cfg.WireGuardConfigPath, cfg.TunnelIP, cfg.PrivateKey, wireguardInterface, wireguardListenPort, nil)
	} else {
		netConf = wireguard.NewNoOpConfigurer()
	}

	err = netConf.SetupInterface()
	if err != nil {
		return fmt.Errorf("set up network interface: %w", err)
	}

	apiserver := &pb.Gateway{
		Name:      "apiserver",
		PublicKey: cfg.APIServerPublicKey,
		Endpoint:  cfg.APIServerEndpoint,
		Ip:        cfg.APIServerTunnelIP,
	}

	// apply initial base config
	err = netConf.ApplyWireGuardConfig([]wireguard.Peer{apiserver})
	if err != nil {
		return fmt.Errorf("apply initial WireGuard config: %w", err)
	}

	grpcClient, err := grpc.DialContext(ctx, cfg.APIServerURL)
	if err != nil {
		return fmt.Errorf("grpc dial: %w", err)
	}
	_ = grpcClient

	// TODO:
	// register prometheus grpc client
	// get gateways stream
	// on change:
	//   prometheusagent.UpdateConfiguration
	//   lastSuccessfulConfigUpdate.SetToCurrentTime()
	return nil
}
