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

type ErrGRPCConnection error

type NetworkConfigurer interface {
	ApplyWireGuardConfig(devices []*pb.Device) error
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
		return fmt.Errorf("connect to api server: %w", err)
	}

	log.Infof("Connected to API server")

	defer apiserver.Close()

	apiserverClient := pb.NewAPIServerClient(apiserver)

	stream, err := apiserverClient.GetGatewayConfiguration(ctx, &pb.GetGatewayConfigurationRequest{
		Gateway:  config.Name,
		Password: config.APIServerPassword,
	})

	if err != nil {
		return err
	}

	log.Infof("Authenticated with API server and streaming configuration updates.")

	for {
		gwConfig, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("get gateway config: %w", err)
		}

		log.Infof("Received updated configuration.")

		err = applyGatewayConfig(netConf, gwConfig)
		if err != nil {
			return fmt.Errorf("apply gateway config: %w", err)
		}
	}
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

	err = configurer.ApplyWireGuardConfig(gatewayConfig.Devices)
	if err != nil {
		return fmt.Errorf("actuating WireGuard config: %w", err)
	}

	err = configurer.ForwardRoutes(gatewayConfig.Routes)
	if err != nil {
		return fmt.Errorf("forwarding routes: %w", err)
	}

	return nil
}
