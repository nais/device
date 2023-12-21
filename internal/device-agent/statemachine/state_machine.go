package statemachine

import (
	"context"
	"fmt"
	"time"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/notify"
	"github.com/sirupsen/logrus"

	"github.com/nais/device/internal/pb"
)

type Event string

const (
	EventWaitForExternalEvent Event = "WaitForExternalEvent"
	EventLogin                Event = "Login"
	EventAuthenticated        Event = "Authenticated"
	EventBootstrapped         Event = "Bootstrapped"
	EventDisconnect           Event = "Disconnect"
)

const (
	helperTimeout = 20 * time.Second
)

type baseState struct {
	rc         runtimeconfig.RuntimeConfig
	cfg        config.Config
	logger     logrus.FieldLogger
	baseStatus *pb.AgentStatus
	notifier   notify.Notifier
}

type State interface {
	Enter(context.Context) Event
	AgentState() pb.AgentState
	String() string
	Status() *pb.AgentStatus
}

type StateMachine struct {
	ctx           context.Context
	current       *stateLifecycle
	events        chan Event
	initialState  State
	transitions   map[Event]transitions
	logger        logrus.FieldLogger
	statusUpdates chan<- *pb.AgentStatus
}

type transitions struct {
	state   State
	sources []State
}

func NewStateMachine(
	ctx context.Context,
	rc runtimeconfig.RuntimeConfig,
	cfg config.Config,
	notifier notify.Notifier,
	deviceHelper pb.DeviceHelperClient,
	statusUpdates chan<- *pb.AgentStatus,
	logger logrus.FieldLogger,
) *StateMachine {
	stateMachine := StateMachine{
		ctx:           ctx,
		events:        make(chan Event, 255),
		logger:        logger,
		statusUpdates: statusUpdates,
	}

	baseState := baseState{
		rc:       rc,
		cfg:      cfg,
		notifier: notifier,
		logger:   logger,
	}

	stateDisconnected := &Disconnected{
		baseState: baseState,
	}

	stateAuthenticating := &Authenticating{
		baseState: baseState,
	}

	stateBootstrapping := &Bootstrapping{
		baseState:    baseState,
		deviceHelper: deviceHelper,
	}

	stateConnected := &Connected{
		baseState:           baseState,
		deviceHelper:        deviceHelper,
		triggerStatusUpdate: stateMachine.TriggerStatusUpdate,
	}

	stateMachine.transitions = map[Event]transitions{
		EventLogin: {
			state: stateAuthenticating,
			sources: []State{
				stateDisconnected,
			},
		},
		EventAuthenticated: {
			state: stateBootstrapping,
			sources: []State{
				stateAuthenticating,
			},
		},
		EventBootstrapped: {
			state: stateConnected,
			sources: []State{
				stateBootstrapping,
			},
		},
		EventDisconnect: {
			state: stateDisconnected,
			sources: []State{
				stateConnected,
				stateAuthenticating,
				stateBootstrapping,
			},
		},
	}

	stateMachine.initialState = stateDisconnected

	for e, t := range stateMachine.transitions {
		if t.state == nil {
			panic(fmt.Sprintf("transition with nil state detected for event: %v", e))
		}
		for _, s := range t.sources {
			if s == nil {
				panic(fmt.Sprintf("transition with nil source detected for event: %v", e))
			}
		}
	}

	return &stateMachine
}

func (sm *StateMachine) TriggerStatusUpdate() {
	select {
	case sm.statusUpdates <- sm.current.state.Status():
	default:
	}
}

func (sm *StateMachine) SendEvent(e Event) {
	sm.events <- e
}

func (sm *StateMachine) Run(ctx context.Context) {
	sm.setState(sm.initialState)

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():

		case event := <-sm.events:
			if ctx.Err() != nil {
				break
			}

			sm.logger.Infof("Event received: %s", event)
			if event == EventWaitForExternalEvent {
				continue
			}

			sm.transition(event)
		}
	}

	sm.current.exit()
}

func (sm *StateMachine) GetAgentState() pb.AgentState {
	if sm.current == nil {
		return sm.initialState.AgentState()
	}

	return sm.current.state.AgentState()
}

func (sm *StateMachine) setState(state State) {
	sm.logger.Infof("Exiting state: %v", sm.current)
	sm.current.exit()

	sm.current = newStateLifecycle(sm.ctx, state)
	sm.TriggerStatusUpdate()

	sm.logger.Infof("Entering state: %v", sm.current)
	sm.current.enter(sm.events)
}

func (sm *StateMachine) transition(event Event) {
	t, ok := sm.transitions[event]
	if !ok {
		sm.logger.Warnf("No defined transitions for event: %s", event)
	}

	for _, s := range t.sources {
		if s == sm.current.state {
			sm.setState(t.state)
			return
		}
	}

	sm.logger.Warnf("No defined transition for event %s in state %s", event, sm.GetAgentState())
}
