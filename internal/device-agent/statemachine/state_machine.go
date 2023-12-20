package statemachine

import (
	"context"
	"fmt"

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

type BaseState struct {
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
	ctx               context.Context
	current           *stateHandle
	events            chan Event
	initialAgentState pb.AgentState
	states            map[pb.AgentState]State
	transitions       map[transitionKey]pb.AgentState
	logger            logrus.FieldLogger
	statusUpdates     chan<- *pb.AgentStatus
}

type transitions struct {
	Event       Event
	Sources     []pb.AgentState
	Destination pb.AgentState
}

type transitionKey struct {
	event  Event
	source State
}

func NewStateMachine(ctx context.Context, rc runtimeconfig.RuntimeConfig, cfg config.Config, notifier notify.Notifier, deviceHelper pb.DeviceHelperClient, statusUpdates chan<- *pb.AgentStatus, logger logrus.FieldLogger) *StateMachine {
	transitions := []transitions{
		{
			Event: EventLogin,
			Sources: []pb.AgentState{
				pb.AgentState_Disconnected,
			},
			Destination: pb.AgentState_Authenticating,
		},
		{
			Event: EventAuthenticated,
			Sources: []pb.AgentState{
				pb.AgentState_Authenticating,
			},
			Destination: pb.AgentState_Bootstrapping,
		},
		{
			Event: EventBootstrapped,
			Sources: []pb.AgentState{
				pb.AgentState_Bootstrapping,
			},
			Destination: pb.AgentState_Connected,
		},
		{
			Event: EventDisconnect,
			Sources: []pb.AgentState{
				pb.AgentState_Connected,
				pb.AgentState_Authenticating,
				pb.AgentState_Bootstrapping,
			},
			Destination: pb.AgentState_Disconnected,
		},
	}

	stateMachine := StateMachine{
		ctx:               ctx,
		events:            make(chan Event, 255),
		states:            make(map[pb.AgentState]State),
		transitions:       make(map[transitionKey]pb.AgentState),
		initialAgentState: pb.AgentState_Disconnected,
		logger:            logger,
		statusUpdates:     statusUpdates,
	}

	baseState := BaseState{
		rc:       rc,
		cfg:      cfg,
		notifier: notifier,
		logger:   logger,
	}

	states := []State{
		&Disconnected{
			BaseState: baseState,
		},
		&Authenticating{
			BaseState: baseState,
		},
		&Bootstrapping{
			BaseState:    baseState,
			deviceHelper: deviceHelper,
		},
		&Connected{
			BaseState:           baseState,
			deviceHelper:        deviceHelper,
			triggerStatusUpdate: stateMachine.TriggerStatusUpdate,
		},
	}

	for _, state := range states {
		stateMachine.states[state.AgentState()] = state
	}

	for _, transition := range transitions {
		if stateMachine.states[transition.Destination] == nil {
			panic(fmt.Sprintf("destination state %s not found", transition.Destination))
		}
		for _, source := range transition.Sources {
			if stateMachine.states[source] == nil {
				panic(fmt.Sprintf("source state %s not found", source))
			}
			stateMachine.transitions[transitionKey{transition.Event, stateMachine.states[source]}] = transition.Destination
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
	sm.setState(sm.initialAgentState)
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
		return sm.initialAgentState
	}

	return sm.current.state.AgentState()
}

func (sm *StateMachine) setState(agentState pb.AgentState) {
	var state State
	state, ok := sm.states[agentState]
	if !ok {
		panic("state not found")
	}

	sm.logger.Infof("Exiting state: %v", sm.current)
	sm.current.exit()

	sm.current = newStateHandle(sm.ctx, state)

	sm.logger.Infof("Entering state: %v", sm.current)
	sm.current.enter(sm.events)
}

func (sm *StateMachine) transition(event Event) {
	key := transitionKey{event, sm.current.state}
	if agentState, ok := sm.transitions[key]; ok {
		sm.setState(agentState)
	} else {
		sm.logger.Warnf("No defined transition for event %s in state %s", event, sm.GetAgentState())
	}
}
