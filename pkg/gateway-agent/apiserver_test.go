package gateway_agent_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nais/device/pkg/gateway-agent"
	"github.com/nais/device/pkg/gateway-agent/config"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/wireguard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSyncFromStream(t *testing.T) {
	const name = "gatewayname"
	const password = "password"

	knownError := errors.New("known error")

	gateway_agent.InitializeMetrics(name, "bar")

	t.Run("control loop runs all the relevant bits, then wraps around", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		req := &pb.GetGatewayConfigurationRequest{
			Gateway:  name,
			Password: password,
		}
		resp := &pb.GetGatewayConfigurationResponse{
			Devices:    []*pb.Device{},
			RoutesIPv4: []string{},
		}
		cfg := config.Config{
			Name:              name,
			APIServerPassword: password,
		}

		stream := &pb.MockAPIServer_GetGatewayConfigurationClient{}
		stream.On("Recv").Return(resp, nil).Once()
		stream.On("Recv").Return(nil, knownError).Once()

		client := &pb.MockAPIServerClient{}
		client.On("GetGatewayConfiguration",
			mock.Anything,
			req,
		).Return(stream, nil)

		staticPeers := cfg.StaticPeers()
		peers := wireguard.CastPeerList(resp.Devices)
		peers = append(peers, staticPeers...)
		netConf := &wireguard.MockNetworkConfigurer{}
		netConf.On("ConnectedDeviceCount").Return(1, nil)
		netConf.On("ApplyWireGuardConfig", peers).Return(nil)
		netConf.On("ForwardRoutesV4", resp.GetRoutesIPv4()).Return(nil)
		netConf.On("ForwardRoutesV6", resp.GetRoutesIPv6()).Return(nil)

		err := gateway_agent.SyncFromStream(ctx, cfg.Name, cfg.APIServerPassword, staticPeers, client, netConf)

		assert.ErrorIs(t, err, knownError)
	})
}
