package device_agent

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine"
	"github.com/nais/device/internal/device-agent/states/authenticating"
	"github.com/nais/device/internal/device-agent/states/bootstrapping"
	"github.com/nais/device/internal/device-agent/states/connected"
	"github.com/nais/device/internal/device-agent/states/disconnected"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/pb"
)

func NewStateMachine(
	ctx context.Context,
	rc runtimeconfig.RuntimeConfig,
	cfg config.Config,
	notifier notify.Notifier,
	deviceHelper pb.DeviceHelperClient,
	statusUpdates chan<- *pb.AgentStatus,
	logger logrus.FieldLogger,
) *statemachine.StateMachine {
	stateDisconnected := disconnected.New(rc, cfg)
	stateAuthenticating := authenticating.New(rc, cfg, logger, notifier)
	stateBootstrapping := bootstrapping.New(rc, logger, notifier, deviceHelper)
	stateConnected := connected.New(rc, logger, notifier, deviceHelper, statusUpdates)

	transitions := map[statemachine.EventType]statemachine.Transitions{
		statemachine.EventLogin: {
			Target: stateAuthenticating,
			Sources: []statemachine.State{
				stateDisconnected,
			},
		},
		statemachine.EventAuthenticated: {
			Target: stateBootstrapping,
			Sources: []statemachine.State{
				stateAuthenticating,
			},
		},
		statemachine.EventBootstrapped: {
			Target: stateConnected,
			Sources: []statemachine.State{
				stateBootstrapping,
			},
		},
		statemachine.EventDisconnect: {
			Target: stateDisconnected,
			Sources: []statemachine.State{
				stateConnected,
				stateAuthenticating,
				stateBootstrapping,
			},
		},
	}

	return statemachine.New(ctx, transitions, stateDisconnected, statusUpdates, logger)
}
