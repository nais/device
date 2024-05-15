package disconnected_test

import (
	"context"
	"testing"

	"github.com/nais/device/internal/device-agent/auth"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine"
	"github.com/nais/device/internal/device-agent/states/disconnected"
	"github.com/nais/device/internal/pb"
	"github.com/stretchr/testify/assert"
)

func TestDisconnected(t *testing.T) {
	t.Run("disconnected waits for more events", func(t *testing.T) {
		rc := runtimeconfig.NewMockRuntimeConfig(t)
		var token *auth.Tokens
		rc.EXPECT().SetToken(token)
		rc.EXPECT().ResetEnrollConfig()

		cfg := config.Config{
			AgentConfiguration: &pb.AgentConfiguration{
				AutoConnect: false,
			},
		}
		state := disconnected.New(rc, cfg)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		event := state.Enter(ctx).Event
		assert.Equal(t, statemachine.EventWaitForExternalEvent, event)
	})

	t.Run("disconnected auto logins if configured (once)", func(t *testing.T) {
		rc := runtimeconfig.NewMockRuntimeConfig(t)
		var token *auth.Tokens
		rc.EXPECT().SetToken(token)
		rc.EXPECT().ResetEnrollConfig()

		cfg := config.Config{
			AgentConfiguration: &pb.AgentConfiguration{
				AutoConnect: true,
			},
		}
		state := disconnected.New(rc, cfg)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		assert.Equal(t, statemachine.EventLogin, state.Enter(ctx).Event)
		assert.Equal(t, statemachine.EventWaitForExternalEvent, state.Enter(ctx).Event)
	})

}
