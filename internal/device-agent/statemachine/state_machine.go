package statemachine

import (
	"context"
	"fmt"
	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/notify"
	"github.com/sirupsen/logrus"
	"sync"

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

type State interface {
	Enter(context.Context) Event
	AgentState() pb.AgentState
	String() string
}

type StateMachine struct {
	ctx          context.Context
	current      *stateHandle
	events       chan Event
	initialState pb.AgentState
	states       map[pb.AgentState]State
	transitions  map[transitionKey]pb.AgentState
	logger       logrus.FieldLogger
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

func NewStateMachine(ctx context.Context, rc runtimeconfig.RuntimeConfig, cfg config.Config, notifier notify.Notifier, deviceHelper pb.DeviceHelperClient, logger logrus.FieldLogger) *StateMachine {
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
	states := []State{
		&Disconnected{
			rc:  rc,
			cfg: cfg,
		},
		&Authenticating{
			rc:       rc,
			cfg:      cfg,
			notifier: notifier,
			logger:   logger,
		},
		&Bootstrapping{
			rc:           rc,
			cfg:          cfg,
			notifier:     notifier,
			deviceHelper: deviceHelper,
			logger:       logger,
		},
		&Connected{
			rc:           rc,
			cfg:          cfg,
			notifier:     notifier,
			deviceHelper: deviceHelper,
			logger:       logger,
		},
	}

	stateMachine := StateMachine{
		ctx:          ctx,
		events:       make(chan Event, 255),
		states:       make(map[pb.AgentState]State),
		transitions:  make(map[transitionKey]pb.AgentState),
		initialState: pb.AgentState_Disconnected,
		logger:       logger,
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
			if ctx.Err() == nil {
				sm.logger.Infof("Event received: %s", event)
				sm.transition(event)
			}
		}
	}
}

func (sm *StateMachine) GetAgentState() pb.AgentState {
	if sm.current == nil {
		return sm.initialState
	}

	for agentState, state := range sm.states {
		if state == sm.current.state {
			return agentState
		}
	}

	panic("current state does not exist")
}

func (sm *StateMachine) setState(agentState pb.AgentState) {
	var state State
	state, ok := sm.states[agentState]
	if !ok {
		panic("state not found")
	}

	if sm.current != nil {
		if sm.current.cancelFunc != nil {
			sm.current.cancelFunc()
			sm.current.mutex.Lock()
		} else {
			panic("Current state has no cancel function, this is a programmer error")
		}
	}

	stateCtx, stateCancel := context.WithCancel(sm.ctx)
	sm.current = &stateHandle{
		state:      state,
		cancelFunc: stateCancel,
		mutex:      &sync.Mutex{},
	}
	sm.logger.Infof("Entering state: %v", state)
	sm.current.mutex.Lock()
	go func() {
		maybeEvent := sm.current.state.Enter(stateCtx)
		if maybeEvent != EventWaitForExternalEvent {
			sm.events <- maybeEvent
		}
		sm.current.mutex.Unlock()
	}()
}

func (sm *StateMachine) transition(event Event) {
	key := transitionKey{event, sm.current.state}
	if agentState, ok := sm.transitions[key]; ok {
		sm.setState(agentState)
	} else {
		sm.logger.Warnf("No defined transition for event %s in state %s", event, sm.GetAgentState())
	}
}

type stateHandle struct {
	state      State
	cancelFunc context.CancelFunc
	mutex      *sync.Mutex
}
