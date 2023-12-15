package statemachine

import (
	"context"
	"testing"
	"time"

	"github.com/nais/device/internal/pb"
	"github.com/stretchr/testify/assert"
)

func TestStateMachine(t *testing.T) {
	t.Run("Check happy path states", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		sm := NewStateMachine(ctx)
		go sm.Run(ctx)

		sm.SendEvent(EventLogin)
		assert.Eventually(t, func() bool { return sm.GetAgentState() == pb.AgentState_Connected }, 100*time.Millisecond, 5*time.Millisecond)
		sm.SendEvent(EventDisconnect)
		assert.Eventually(t, func() bool { return sm.GetAgentState() == pb.AgentState_Disconnected }, 100*time.Millisecond, 5*time.Millisecond)
	})
}
