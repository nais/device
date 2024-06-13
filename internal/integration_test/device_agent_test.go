package integrationtest_test

import (
	"context"
	"sync"
	"testing"

	device_agent "github.com/nais/device/internal/device-agent"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func NewDeviceAgent(t *testing.T, wg *sync.WaitGroup, ctx context.Context, log *logrus.Entry, helperconn *bufconn.Listener, rc *runtimeconfig.MockRuntimeConfig) *grpc.Server {
	helperDial, err := dial(helperconn)
	assert.NoError(t, err)

	helperClient := pb.NewDeviceHelperClient(helperDial)

	cfg, err := config.DefaultConfig()
	assert.NoError(t, err)

	cfg.AgentConfiguration = &pb.AgentConfiguration{}
	cfg.LogLevel = logrus.DebugLevel.String()

	notifier := notify.NewMockNotifier(t)
	notifier.EXPECT().Errorf(mock.Anything, mock.Anything).Maybe()
	notifier.EXPECT().Infof(mock.Anything, mock.Anything).Maybe()

	statusChannel := make(chan *pb.AgentStatus, 32)
	stateMachine := device_agent.NewStateMachine(ctx, rc, *cfg, notifier, helperClient, statusChannel, log)

	impl := device_agent.NewServer(ctx, log, cfg, rc, notifier, stateMachine.SendEvent)
	server := grpc.NewServer()
	pb.RegisterDeviceAgentServer(server, impl)

	wg.Add(1)
	go func() {
		stateMachine.Run(ctx)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		// This routine forwards status updates from the state machine to the device agent server
		for ctx.Err() == nil {
			select {
			case s := <-statusChannel:
				impl.UpdateAgentStatus(s)
			case <-ctx.Done():
			}
		}
		wg.Done()
	}()

	return server
}
