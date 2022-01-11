package gateway_agent

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/go-iptables/iptables"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const syncConfigDialTimeout = 1 * time.Second

type NetworkConfigurer interface {
	ActuateWireGuardConfig(devices []*pb.Device) error
	ForwardRoutes(routes []string) error
	ConnectedDeviceCount() (int, error)
	SetupInterface() error
	SetupIPTables() error
}

type networkConfigurer struct {
	config        Config
	ipTables      *iptables.IPTables
	interfaceName string
	interfaceIP   string
}

func NewConfigurer(config Config, ipTables *iptables.IPTables) NetworkConfigurer {
	return &networkConfigurer{
		config:   config,
		ipTables: ipTables,
	}
}

func SyncFromStream(ctx context.Context, config Config, netConf NetworkConfigurer) error {
	stream, err := setupGatewayConfigStream(ctx, config)
	if err != nil {
		return fmt.Errorf("connecting to gateway config stream: %w", err)
	}
	for {
		gwConfig, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("get gateway config: %w", err)
		}
		applyGatewayConfig(netConf, gwConfig)
	}
}

func setupGatewayConfigStream(ctx context.Context, config Config) (pb.APIServer_GetGatewayConfigurationClient, error) {
	dialContext, cancel := context.WithTimeout(ctx, syncConfigDialTimeout)
	defer cancel()

	log.Infof("Attempting gRPC connection to API server on %s...", config.APIServerURL)
	apiserver, err := grpc.DialContext(
		dialContext,
		config.APIServerURL,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
	)

	if err != nil {
		return nil, fmt.Errorf("connect to api server: %w", err)
	}

	defer apiserver.Close()

	apiserverClient := pb.NewAPIServerClient(apiserver)

	return apiserverClient.GetGatewayConfiguration(ctx, &pb.GetGatewayConfigurationRequest{
		Gateway:  config.Name,
		Password: config.APIServerPassword,
	})
}

func applyGatewayConfig(configurer NetworkConfigurer, gatewayConfig *pb.GetGatewayConfigurationResponse) error {
	RegisteredDevices.Set(float64(len(gatewayConfig.Devices)))
	LastSuccessfulConfigFetch.SetToCurrentTime()

	c, err := configurer.ConnectedDeviceCount()
	if err != nil {
		log.Errorf("getting connected device count: %v", err)
	} else {
		ConnectedDevices.Set(float64(c))
	}

	err = configurer.ActuateWireGuardConfig(gatewayConfig.Devices)
	if err != nil {
		return fmt.Errorf("actuating WireGuard config: %w", err)
	}

	err = configurer.ForwardRoutes(gatewayConfig.Routes)
	if err != nil {
		return fmt.Errorf("forwarding routes: %w", err)
	}

	return nil
}
