package integrationtest_test

import (
	"context"
	"testing"

	gateway_agent "github.com/nais/device/internal/gateway-agent"
	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/test/bufconn"
)

func StartGatewayAgent(t *testing.T, ctx context.Context, log *logrus.Entry, name string, apiserverConn *bufconn.Listener, apiserverPeer *pb.Gateway, networkConfigurer wireguard.NetworkConfigurer) error {
	apiserverDial, err := dial(apiserverConn)
	assert.NoError(t, err)

	apiserverClient := pb.NewAPIServerClient(apiserverDial)

	return gateway_agent.SyncFromStream(ctx, log, name, "password", []wireguard.Peer{apiserverPeer}, apiserverClient, networkConfigurer)
}
