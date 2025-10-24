package deviceagent

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/nais/device/internal/deviceagent/auth"
	"github.com/nais/device/internal/deviceagent/config"
	"github.com/nais/device/internal/deviceagent/runtimeconfig"
	"github.com/nais/device/internal/deviceagent/statemachine"
	"github.com/nais/device/internal/deviceagent/statemachine/state"
	"github.com/nais/device/internal/deviceagent/states/authenticating"
	"github.com/nais/device/internal/deviceagent/states/bootstrapping"
	"github.com/nais/device/internal/deviceagent/states/connected"
	"github.com/nais/device/internal/deviceagent/states/disconnected"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/pkg/pb"
)

func NewStateMachine(
	ctx context.Context,
	rc runtimeconfig.RuntimeConfig,
	cfg config.Config,
	notifier notify.Notifier,
	deviceHelper pb.DeviceHelperClient,
	statusUpdates chan<- *pb.AgentStatus,
	authHandler auth.Handler,
	logger logrus.FieldLogger,
) *statemachine.StateMachine {
	stateDisconnected := disconnected.New(rc, cfg)
	stateAuthenticating := authenticating.New(rc, cfg, authHandler, logger, notifier)
	stateBootstrapping := bootstrapping.New(rc, logger, notifier, deviceHelper)
	stateConnected := connected.New(rc, logger, notifier, deviceHelper, statusUpdates)

	transitions := map[state.EventType]statemachine.Transitions{
		state.EventLogin: {
			Target: stateAuthenticating,
			Sources: []state.State{
				stateDisconnected,
			},
		},
		state.EventAuthenticated: {
			Target: stateBootstrapping,
			Sources: []state.State{
				stateAuthenticating,
			},
		},
		state.EventBootstrapped: {
			Target: stateConnected,
			Sources: []state.State{
				stateBootstrapping,
			},
		},
		state.EventDisconnect: {
			Target: stateDisconnected,
			Sources: []state.State{
				stateConnected,
				stateAuthenticating,
				stateBootstrapping,
			},
		},
	}

	return statemachine.New(ctx, transitions, stateDisconnected, statusUpdates, logger)
}
