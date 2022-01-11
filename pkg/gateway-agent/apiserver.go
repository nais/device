package gateway_agent

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const syncConfigDialTimeout = 1 * time.Second

type NetworkConfigurer interface {
	ActuateWireGuardConfig(devices []*pb.Device) error
	ForwardRoutes(routes []string) error
}

type networkConfigurer struct {
	config Config
}

func NewConfigurer(config Config) networkConfigurer {
	return networkConfigurer{
		config: config,
	}
}

func ApplyGatewayConfig(configurer networkConfigurer, gatewayConfig *pb.GetGatewayConfigurationResponse) {

	RegisteredDevices.Set(float64(len(gatewayConfig.Devices)))

	LastSuccessfulConfigFetch.SetToCurrentTime()
	log.Debugf("%+v\n", gatewayConfig)
	// skip side-effects for local development
	if configurer.config.DevMode {
		return
	}
	if c, err := ConnectedDeviceCount(); err != nil {
		log.Errorf("Getting connected device count: %v", err)
	} else {
		ConnectedDevices.Set(float64(c))
	}

	err := configurer.ActuateWireGuardConfig(gatewayConfig.Devices)
	if err != nil {
		log.Errorf("actuating WireGuard config: %v", err)
	}

	err = configurer.ForwardRoutes(gatewayConfig.Routes)
	if err != nil {
		log.Errorf("forwarding routes: %v", err)
	}
}

func GetGatewayConfig(ctx context.Context, config Config) (pb.APIServer_GetGatewayConfigurationClient, error) {
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
