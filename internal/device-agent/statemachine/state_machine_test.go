package statemachine

import (
	"context"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/notify"
	"github.com/sirupsen/logrus"
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

		cfg := config.Config{}
		notifier := notify.NewMockNotifier(t)

		sm := NewStateMachine(ctx, rc, cfg, notifier, logrus.New())
		go sm.Run(ctx)

		sm.SendEvent(EventLogin)
		assert.Eventually(t, func() bool { return sm.GetAgentState() == pb.AgentState_Connected }, 100*time.Millisecond, 5*time.Millisecond)
		sm.SendEvent(EventDisconnect)
		assert.Eventually(t, func() bool { return sm.GetAgentState() == pb.AgentState_Disconnected }, 100*time.Millisecond, 5*time.Millisecond)
	})
}
