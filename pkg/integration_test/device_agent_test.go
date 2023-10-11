package integrationtest_test

import (
	"context"
	"testing"

	device_agent "github.com/nais/device/pkg/device-agent"
	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func NewDeviceAgent(t *testing.T, ctx context.Context, helperconn, apiconn *bufconn.Listener) *grpc.Server {
	apiDial, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(ContextBufDialer(apiconn)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	assert.NoError(t, err)

	helperDial, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(ContextBufDialer(helperconn)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	assert.NoError(t, err)
	helperClient := pb.NewDeviceHelperClient(helperDial)

	cfg := config.DefaultConfig()
	rc := runtimeconfig.NewMockRuntimeConfig(t)
	rc.EXPECT().DialAPIServer(mock.Anything).Return(apiDial, nil).Once()

	assert.NoError(t, err)

	impl := device_agent.NewServer(helperClient, &cfg, rc)

	server := grpc.NewServer()
	pb.RegisterDeviceAgentServer(server, impl)

	return server
}
