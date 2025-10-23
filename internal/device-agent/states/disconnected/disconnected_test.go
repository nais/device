package disconnected_test

import (
	"context"
	"testing"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	statemachine "github.com/nais/device/internal/device-agent/statemachine/state"
	"github.com/nais/device/internal/device-agent/states/disconnected"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
)

func TestDisconnected(t *testing.T) {
	t.Run("disconnected waits for more events", func(t *testing.T) {
		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().ResetEnrollConfig()
		rc.EXPECT().SetAPIServerInfo(nil, "").Return()

		cfg := config.Config{
			AgentConfiguration: &pb.AgentConfiguration{
				AutoConnect: false,
			},
		}
		stateDisconnected := disconnected.New(rc, cfg)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		event := stateDisconnected.Enter(ctx).Event
		assert.Equal(t, statemachine.EventWaitForExternalEvent, event)
	})

	t.Run("disconnected auto logins if configured (once)", func(t *testing.T) {
		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().ResetEnrollConfig()
		rc.EXPECT().SetAPIServerInfo(nil, "").Return()

		cfg := config.Config{
			AgentConfiguration: &pb.AgentConfiguration{
				AutoConnect: true,
			},
		}
		stateDisconnected := disconnected.New(rc, cfg)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		assert.Equal(t, statemachine.EventLogin, stateDisconnected.Enter(ctx).Event)
		assert.Equal(t, statemachine.EventWaitForExternalEvent, stateDisconnected.Enter(ctx).Event)
	})
}
