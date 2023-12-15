package statemachine

import (
	"context"
	"fmt"

	"github.com/nais/device/internal/pb"
)

type Event int8

const (
	EventLogin Event = iota
	EventAuthenticated
	EventBootstrapped
	EventDisconnect
)

type State interface {
	Enter(context.Context, func(Event))
}

type StateMachine struct {
	ctx          context.Context
	current      *stateHandle
	events       chan Event
	initialState pb.AgentState
	states       map[pb.AgentState]State
	transitions  map[transitionKey]pb.AgentState
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

func NewStateMachine(ctx context.Context) *StateMachine {
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
		ctx:    ctx,
		events: make(chan Event, 255),
		states: map[pb.AgentState]State{
			pb.AgentState_Disconnected:   &Disconnected{},
			pb.AgentState_Authenticating: &Authenticating{},
			pb.AgentState_Bootstrapping:  &Bootstrapping{},
			pb.AgentState_Connected:      &Connected{},
		},
		transitions:  make(map[transitionKey]pb.AgentState),
		initialState: pb.AgentState_Disconnected,
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
	for {
		select {
		case event := <-sm.events:
			sm.transition(event)

		case <-ctx.Done():
			return
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
		} else {
			// TODO: BUG, log it
		}
	}

	stateCtx, stateCancel := context.WithCancel(sm.ctx)
	sm.current = &stateHandle{
		state:      state,
		cancelFunc: stateCancel,
	}
	go sm.current.state.Enter(stateCtx, sm.SendEvent)
}

func (sm *StateMachine) transition(event Event) {
	key := transitionKey{event, sm.current.state}
	if agentState, ok := sm.transitions[key]; ok {
		sm.setState(agentState)
	}
}

type stateHandle struct {
	state      State
	cancelFunc context.CancelFunc
}
