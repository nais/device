package statemachine

import (
	"context"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/notify"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
	"time"

	"github.com/nais/device/internal/pb"
	"github.com/stretchr/testify/assert"
)

func TestStateMachine(t *testing.T) {
	t.Run("Check happy path states", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().GetTenantSession().Return(&pb.Session{
			Expiry: timestamppb.New(time.Now().Add(time.Hour)),
		}, nil)
		rc.EXPECT().LoadEnrollConfig().Return(nil)
		rc.EXPECT().APIServerPeer().Return(&pb.Gateway{})
		rc.EXPECT().BuildHelperConfiguration(mock.Anything).Return(&pb.Configuration{})
		rc.EXPECT().DialAPIServer(mock.Anything).Return(&grpc.ClientConn{}, nil)

		cfg := config.Config{}

		notifier := notify.NewMockNotifier(t)

		deviceHelper := pb.NewMockDeviceHelperClient(t)
		deviceHelper.EXPECT().Configure(mock.Anything, mock.Anything).Return(&pb.ConfigureResponse{}, nil)

		sm := NewStateMachine(ctx, rc, cfg, notifier, deviceHelper, logrus.New())
		go sm.Run(ctx)

		sm.SendEvent(EventLogin)
		assert.Eventually(t, func() bool { return sm.GetAgentState() == pb.AgentState_Connected }, 100*time.Millisecond, 5*time.Millisecond)
		sm.SendEvent(EventDisconnect)
		assert.Eventually(t, func() bool { return sm.GetAgentState() == pb.AgentState_Disconnected }, 100*time.Millisecond, 5*time.Millisecond)
	})
}
