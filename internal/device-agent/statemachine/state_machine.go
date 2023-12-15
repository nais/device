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
	Enter(ctx context.Context, sendEvent func(Event))
	Exit()
	GetAgentState() pb.AgentState
}

type StateMachine struct {
	stateChanges chan State
	ctx          context.Context
	cancelFunc   context.CancelFunc
	states       map[pb.AgentState]State
	transitions  map[transitionKey]pb.AgentState
	currentState *stateHandle
}

type Transitions struct {
	Event       Event
	Sources     []pb.AgentState
	Destination pb.AgentState
}

type transitionKey struct {
	event  Event
	source pb.AgentState
}

func NewStateMachine(ctx context.Context) (*StateMachine, error) {
	ctx, cancelFunc := context.WithCancel(ctx)

	initialState := pb.AgentState_Disconnected

	transitions := []Transitions{
		{EventLogin, []pb.AgentState{pb.AgentState_Disconnected}, pb.AgentState_Authenticating},
		{EventAuthenticated, []pb.AgentState{pb.AgentState_Authenticating}, pb.AgentState_Bootstrapping},
		{EventBootstrapped, []pb.AgentState{pb.AgentState_Bootstrapping}, pb.AgentState_Connected},
		{EventDisconnect, []pb.AgentState{pb.AgentState_Connected, pb.AgentState_Authenticating, pb.AgentState_Bootstrapping}, pb.AgentState_Disconnected},
	}

	states := []State{
		&Disconnected{},
		&Authenticating{},
		&Bootstrapping{},
		&Connected{},
	}
	stateMachine := StateMachine{
		ctx:         ctx,
		cancelFunc:  cancelFunc,
		states:      make(map[pb.AgentState]State),
		transitions: make(map[transitionKey]pb.AgentState),
	}
	for _, state := range states {
		stateMachine.states[state.GetAgentState()] = state
	}
	for _, transition := range transitions {
		if stateMachine.states[transition.Destination] == nil {
			return nil, fmt.Errorf("destination state %s not found", transition.Destination)
		}
		for _, source := range transition.Sources {
			if stateMachine.states[source] == nil {
				return nil, fmt.Errorf("source state %s not found", source)
			}
			stateMachine.transitions[transitionKey{transition.Event, source}] = transition.Destination
		}
	}
	stateMachine.currentState = stateMachine.states[initialState]
	stateMachine.currentState.Enter(ctx)
	return &stateMachine, nil
}

func (sm *StateMachine) run(ctx context.Context) {
	select {
	case state := <-sm.stateChanges:
		sm.setState(state.GetAgentState())
	case <-ctx.Done():
		return
	}
}

func (sm *StateMachine) setState(agentState pb.AgentState) {
	var state State
	state, ok := sm.states[agentState]
	if !ok {
		panic("state not found")
	}
	sm.cancelFunc()
	sm.currentState.enter(sm.ctx, sm.Transition)
	sm.currentState = &stateHandle{state: state}
	sm.currentState.exit()
}

func (sm *StateMachine) Transition(event Event) {
	key := transitionKey{event, sm.currentState.state.GetAgentState()}
	if agentState, ok := sm.transitions[key]; ok {
		sm.setState(agentState)
	}
}

type stateHandle struct {
	state      State
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func (sh *stateHandle) enter(ctx context.Context, sendEvent func(Event)) {
	sh.ctx, sh.cancelFunc = context.WithCancel(ctx)
	go sh.state.Enter(ctx, sendEvent)
}

func (sh *stateHandle) exit() {
	sh.cancelFunc()
	sh.state.Exit()
}
