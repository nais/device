package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nais/device/pkg/pb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nais/device/pkg/logger"
	prometheusagent "github.com/nais/device/pkg/prometheus-agent"
	"github.com/nais/device/pkg/prometheus-agent/config"
	"github.com/nais/device/pkg/version"
	"github.com/nais/device/pkg/wireguard"
)

var lastSuccessfulConfigUpdate = prometheus.NewGauge(prometheus.GaugeOpts{
	Name:      "last_successful_config_update",
	Help:      "time since last successful prometheus config update",
	Namespace: "naisdevice",
	Subsystem: "prometheus_agent",
})

const (
	updateInterval      = 5 * time.Minute
	updateTimeout       = 10 * time.Second
	wireguardInterface  = "wg0"
	wireguardListenPort = 51820
)

func main() {
	cfg := config.DefaultConfig()
	err := run(cfg)
	if err != nil {
		log.Fatalf("Running prometheus-agent: %s", err)
	}
}

func run(cfg config.Config) error {
	var err error

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = envconfig.Process("PROMETHEUS_AGENT", &cfg)
	if err != nil {
		return fmt.Errorf("read environment configuration: %w", err)
	}

	err = cfg.Parse()
	if err != nil {
		return fmt.Errorf("parse configuration: %w", err)
	}

	logger.Setup(cfg.LogLevel)

	log.Infof("prometheus-agent version %s, revision: %s", version.Version, version.Revision)

	prometheus.MustRegister(lastSuccessfulConfigUpdate)

	go func() {
		log.Infof("Prometheus serving metrics at %v", cfg.PrometheusAddress)
		_ = http.ListenAndServe(cfg.PrometheusAddress, promhttp.Handler())
	}()

	var netConf wireguard.NetworkConfigurer
	if cfg.WireGuardEnabled {
		err = cfg.ValidateWireGuard()
		if err != nil {
			return fmt.Errorf("cannot enable WireGuard: %w", err)
		}

		netConf = wireguard.NewConfigurer(cfg.WireGuardConfigPath, cfg.WireGuardIPv4, cfg.WireGuardIPv6, cfg.PrivateKey, wireguardInterface, wireguardListenPort, nil)
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
		Ipv4:      cfg.APIServerTunnelIP,
	}

	// apply initial base config
	err = netConf.ApplyWireGuardConfig([]wireguard.Peer{apiserver})
	if err != nil {
		return fmt.Errorf("apply initial WireGuard config: %w", err)
	}

	grpcClient, err := grpc.DialContext(ctx, cfg.APIServerURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("grpc dial: %w", err)
	}

	apiClient := pb.NewAPIServerClient(grpcClient)

	ctx, cancel = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	update := func() error {
		ctx, cancel := context.WithTimeout(ctx, updateTimeout)
		defer cancel()

		gateways, err := listGateways(ctx, cfg, apiClient)
		if err != nil {
			return err
		}

		return applyGateways(netConf, gateways, apiserver)
	}

	for ctx.Err() == nil {
		log.Debugf("Polling for new configuration...")
		err = update()
		if err != nil {
			log.Error(err)
		} else {
			log.Debugf("Configuration successfully applied")
			lastSuccessfulConfigUpdate.SetToCurrentTime()
		}

		select {
		case <-ctx.Done():
			log.Infof("prometheus-agent program context done, exiting")
			return nil
		case <-time.After(updateInterval):
		}
	}

	return nil
}

func listGateways(ctx context.Context, cfg config.Config, client pb.APIServerClient) ([]*pb.Gateway, error) {
	const listCap = 128

	stream, err := client.ListGateways(ctx, &pb.ListGatewayRequest{
		Username: cfg.APIServerUsername,
		Password: cfg.APIServerPassword,
	})
	if err != nil {
		return nil, err
	}

	gateways := make([]*pb.Gateway, 0, listCap)

	for ctx.Err() == nil {
		gateway, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		gateways = append(gateways, gateway)
	}

	return gateways, nil
}

func applyGateways(netConf wireguard.NetworkConfigurer, gateways []*pb.Gateway, staticPeers ...wireguard.Peer) error {
	peers := make([]wireguard.Peer, len(gateways))
	ips := make([]string, len(gateways))

	for i := range gateways {
		peers[i] = gateways[i]
		ips[i] = gateways[i].GetIpv4()
	}

	peers = append(peers, staticPeers...)

	err := netConf.ApplyWireGuardConfig(peers)
	if err != nil {
		return err
	}

	return prometheusagent.UpdateConfiguration(ips)
}
