package gateway_agent

import (
	"context"
	"fmt"

	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/internal/wireguard"
	"github.com/sirupsen/logrus"
)

type ErrGRPCConnection error

func SyncFromStream(ctx context.Context, log *logrus.Entry, name, password string, staticPeers []wireguard.Peer, apiserverClient pb.APIServerClient, netConf wireguard.NetworkConfigurer) error {
	stream, err := apiserverClient.GetGatewayConfiguration(ctx, &pb.GetGatewayConfigurationRequest{
		Gateway:  name,
		Password: password,
	})
	if err != nil {
		return err
	}

	log.Info("authenticated with API server and streaming configuration updates")

	for {
		gwConfig, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("get gateway config: %w", err)
		}

		log.Info("received updated configuration")

		err = applyGatewayConfig(netConf, gwConfig, staticPeers...)
		if err != nil {
			return fmt.Errorf("apply gateway config: %w", err)
		}
	}
}

func applyGatewayConfig(configurer wireguard.NetworkConfigurer, gatewayConfig *pb.GetGatewayConfigurationResponse, staticPeers ...wireguard.Peer) error {
	// TODO make struct for gateway runner, and mock this out properly for testing
	// RegisteredDevices.Set(float64(len(gatewayConfig.Devices)))
	// LastSuccessfulConfigFetch.SetToCurrentTime()

	peers := wireguard.CastPeerList(gatewayConfig.Devices)
	err := configurer.ApplyWireGuardConfig(append(staticPeers, peers...))
	if err != nil {
		return fmt.Errorf("actuating WireGuard config: %w", err)
	}

	err = configurer.ForwardRoutesV4(gatewayConfig.GetRoutesIPv4())
	if err != nil {
		return fmt.Errorf("forwarding IPv4 routes: %w", err)
	}

	err = configurer.ForwardRoutesV6(gatewayConfig.GetRoutesIPv6())
	if err != nil {
		return fmt.Errorf("forwarding IPv6 routes: %w", err)
	}

	return nil
}
