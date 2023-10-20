package integrationtest_test

import (
	"context"
	"testing"

	gateway_agent "github.com/nais/device/pkg/gateway-agent"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/wireguard"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/test/bufconn"
)

func StartGatewayAgent(t *testing.T, ctx context.Context, name string, apiserverConn *bufconn.Listener, apiserverPeer *pb.Gateway, networkConfigurer wireguard.NetworkConfigurer) error {
	apiserverDial, err := dial(ctx, apiserverConn)
	assert.NoError(t, err)

	apiserverClient := pb.NewAPIServerClient(apiserverDial)

	return gateway_agent.SyncFromStream(ctx, name, "password", []wireguard.Peer{apiserverPeer}, apiserverClient, networkConfigurer)
}
