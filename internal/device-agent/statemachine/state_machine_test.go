package statemachine

import (
	"context"
	"testing"
	"time"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
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
		mockGetDeviceConfigclient.EXPECT().Recv().WaitUntil(time.After(20*time.Millisecond)).Return(&pb.GetDeviceConfigurationResponse{
			Status:   pb.DeviceConfigurationStatus_DeviceHealthy,
			Gateways: []*pb.Gateway{},
		}, nil)
		mockApiServerClient := pb.NewMockAPIServerClient(t)
		mockApiServerClient.EXPECT().GetDeviceConfiguration(mock.Anything, mock.Anything).Return(mockGetDeviceConfigclient, nil)
		rc.EXPECT().ConnectToAPIServer(mock.Anything).Return(mockApiServerClient, func() {}, nil)
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

		sm := NewStateMachine(ctx, rc, cfg, notifier, deviceHelper, nil, log)
		go sm.Run(ctx)

		sm.SendEvent(EventLogin)
		assert.Eventually(t, func() bool { return sm.GetAgentState() == pb.AgentState_Connected }, 2000*time.Millisecond, 5*time.Millisecond)

		sm.SendEvent(EventDisconnect)
		assert.Eventually(t, func() bool { return sm.GetAgentState() == pb.AgentState_Disconnected }, 3000*time.Millisecond, 5*time.Millisecond)
	})
}
