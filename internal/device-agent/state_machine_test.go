package device_agent_test

import (
	"context"
	"testing"
	"time"

	device_agent "github.com/nais/device/internal/device-agent"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine/state"
	"github.com/nais/device/internal/notify"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/internal/pb"
	"github.com/stretchr/testify/assert"
)

func TestStateMachine(t *testing.T) {
	t.Run("Check happy path states", func(t *testing.T) {
		log := logrus.New()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().GetTenantSession().Return(&pb.Session{
			Key:    "key",
			Expiry: timestamppb.New(time.Now().Add(time.Hour)),
		}, nil)
		rc.EXPECT().LoadEnrollConfig().Return(nil)
		rc.EXPECT().APIServerPeer().Return(&pb.Gateway{})
		rc.EXPECT().BuildHelperConfiguration(mock.Anything).Return(&pb.Configuration{})

		mockGetDeviceConfigclient := pb.NewMockAPIServer_GetDeviceConfigurationClient(t)
		recv := mockGetDeviceConfigclient.EXPECT().Recv().After(20 * time.Millisecond)
		var streamContext context.Context
		recv.Run(func(args mock.Arguments) {
			if streamContext != nil && streamContext.Err() != nil {
				recv.ReturnArguments = []any{nil, streamContext.Err()}
			} else {
				recv.ReturnArguments = []any{
					&pb.GetDeviceConfigurationResponse{
						Status: pb.DeviceConfigurationStatus_DeviceHealthy,
						Gateways: []*pb.Gateway{
							{
								Name: "dummy-gateway",
							},
						},
					}, nil,
				}
			}
		})

		mockApiServerClient := pb.NewMockAPIServerClient(t)
		mockApiServerClient.EXPECT().GetDeviceConfiguration(mock.Anything, mock.Anything).Return(mockGetDeviceConfigclient, nil)
		rc.EXPECT().ConnectToAPIServer(mock.Anything).Return(mockApiServerClient, func() {}, nil).Run(func(ctx context.Context) {
			streamContext = ctx
		})

		rc.EXPECT().SetToken(mock.Anything)
		rc.EXPECT().ResetEnrollConfig()

		cfg := config.Config{
			AgentConfiguration: &pb.AgentConfiguration{},
		}

		notifier := notify.NewMockNotifier(t)
		notifier.EXPECT().Errorf(mock.Anything, mock.Anything).Return().Maybe()

		deviceHelper := pb.NewMockDeviceHelperClient(t)
		deviceHelper.EXPECT().Configure(mock.Anything, mock.Anything).Return(&pb.ConfigureResponse{}, nil)
		deviceHelper.EXPECT().Teardown(mock.Anything, mock.Anything).Return(&pb.TeardownResponse{}, nil)

		statusChan := make(chan *pb.AgentStatus, 10)
		sm := device_agent.NewStateMachine(ctx, rc, cfg, notifier, deviceHelper, statusChan, log)
		go sm.Run(ctx)

		isState := func(state pb.AgentState, numGateways int) func() bool {
			return func() bool {
				select {
				case s := <-statusChan:
					t.Logf("got state: %v", s)
					return state == s.ConnectionState && len(s.Gateways) == numGateways
				default:
					return false
				}
			}
		}

		sm.SendEvent(state.SpanEvent(ctx, state.EventLogin))
		assert.Eventually(t, isState(pb.AgentState_Connected, 1), 2000*time.Millisecond, 5*time.Millisecond)

		sm.SendEvent(state.SpanEvent(ctx, state.EventDisconnect))
		assert.Eventually(t, isState(pb.AgentState_Disconnected, 0), 3000*time.Millisecond, 5*time.Millisecond)
	})
}
