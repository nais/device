package gateway_agent

import (
	"context"
	"fmt"

	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/wireguard"
	log "github.com/sirupsen/logrus"
)

type ErrGRPCConnection error

func SyncFromStream(ctx context.Context, config Config, apiserverClient pb.APIServerClient, netConf wireguard.NetworkConfigurer) error {
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

func applyGatewayConfig(configurer wireguard.NetworkConfigurer, gatewayConfig *pb.GetGatewayConfigurationResponse) error {
	RegisteredDevices.Set(float64(len(gatewayConfig.Devices)))
	LastSuccessfulConfigFetch.SetToCurrentTime()

	c, err := configurer.ConnectedDeviceCount()
	if err != nil {
		log.Errorf("getting connected device count: %v", err)
	} else {
		ConnectedDevices.Set(float64(c))
	}

	err = configurer.ApplyWireGuardConfig(pb.DevicesAsPeers(gatewayConfig.Devices))
	if err != nil {
		return fmt.Errorf("actuating WireGuard config: %w", err)
	}

	err = configurer.ForwardRoutes(gatewayConfig.Routes)
	if err != nil {
		return fmt.Errorf("forwarding routes: %w", err)
	}

	return nil
}
