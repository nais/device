package integrationtest_test

import (
	"context"
	"testing"

	device_agent "github.com/nais/device/pkg/device-agent"
	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func NewDeviceAgent(t *testing.T, ctx context.Context, helperconn *bufconn.Listener, rc *runtimeconfig.MockRuntimeConfig) *grpc.Server {
	helperDial, err := dial(ctx, helperconn)
	assert.NoError(t, err)

	helperClient := pb.NewDeviceHelperClient(helperDial)

	cfg := config.DefaultConfig()
	cfg.AgentConfiguration = &pb.AgentConfiguration{}
	cfg.LogLevel = logrus.DebugLevel.String()

	impl := device_agent.NewServer(helperClient, &cfg, rc)
	go impl.EventLoop(ctx)

	server := grpc.NewServer()
	pb.RegisterDeviceAgentServer(server, impl)

	return server
}
