package integrationtest_test

import (
	"context"
	"sync"
	"testing"

	device_agent "github.com/nais/device/pkg/device-agent"
	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func NewDeviceAgent(t *testing.T, wg *sync.WaitGroup, ctx context.Context, log *logrus.Entry, helperconn *bufconn.Listener, rc *runtimeconfig.MockRuntimeConfig) *grpc.Server {
	helperDial, err := dial(ctx, helperconn)
	assert.NoError(t, err)

	helperClient := pb.NewDeviceHelperClient(helperDial)

	cfg, err := config.DefaultConfig()
	assert.NoError(t, err)

	cfg.AgentConfiguration = &pb.AgentConfiguration{}
	cfg.LogLevel = logrus.DebugLevel.String()

	notifier := notify.NewMockNotifier(t)
	notifier.EXPECT().Errorf(mock.Anything, mock.Anything).Maybe()

	impl := device_agent.NewServer(log, helperClient, cfg, rc, notifier)
	wg.Add(1)
	go func() {
		impl.EventLoop(ctx)
		wg.Done()
	}()

	server := grpc.NewServer()
	pb.RegisterDeviceAgentServer(server, impl)

	return server
}
