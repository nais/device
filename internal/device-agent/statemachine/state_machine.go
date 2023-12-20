package statemachine

import (
	"context"

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
	current       *stateLifeCycle
	events        chan Event
	initialState  State
	transitions   map[Event]transitions
	logger        logrus.FieldLogger
	statusUpdates chan<- *pb.AgentStatus
}

type transitions struct {
	state   State
	sources []pb.AgentState
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
	baseState := baseState{
		rc:       rc,
		cfg:      cfg,
		notifier: notifier,
		logger:   logger,
	}

	stateMachine := StateMachine{
		ctx:           ctx,
		events:        make(chan Event, 255),
		logger:        logger,
		statusUpdates: statusUpdates,
	}

	stateMachine.transitions = map[Event]transitions{
		EventLogin: {
			state: &Authenticating{
				baseState: baseState,
			},
			sources: []pb.AgentState{
				pb.AgentState_Disconnected,
			},
		},
		EventAuthenticated: {
			state: &Bootstrapping{
				baseState:    baseState,
				deviceHelper: deviceHelper,
			},
			sources: []pb.AgentState{
				pb.AgentState_Authenticating,
			},
		},
		EventBootstrapped: {
			state: &Connected{
				baseState:           baseState,
				deviceHelper:        deviceHelper,
				triggerStatusUpdate: stateMachine.TriggerStatusUpdate,
			},
			sources: []pb.AgentState{
				pb.AgentState_Bootstrapping,
			},
		},
		EventDisconnect: {
			state: &Disconnected{
				baseState: baseState,
			},
			sources: []pb.AgentState{
				pb.AgentState_Connected,
				pb.AgentState_Authenticating,
				pb.AgentState_Bootstrapping,
			},
		},
	}

	// hacky, but works i guess
	stateMachine.initialState = stateMachine.transitions[EventDisconnect].state

	// TODO maybe add a validate method here to make sure transitions map does not contain unexpected nil values?

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
			return

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

	sm.logger.Infof("Entering state: %v", sm.current)
	sm.current.enter(sm.events)
}

func (sm *StateMachine) transition(event Event) {
	for e, t := range sm.transitions {
		if e == event {
			sm.setState(t.state)
		}
	}

	sm.logger.Warnf("No defined transition for event %s in state %s", event, sm.GetAgentState())
}
