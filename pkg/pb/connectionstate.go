package pb

// see time.Format for format documentation, equals %H:%M:%S
const timeFormat = "15:04:05"

func (x *AgentStatus) ConnectionStateString() string {
	switch x.ConnectionState {
	case AgentState_Connected:
		return "Connected since " + x.ConnectedSince.AsTime().Format(timeFormat)
	default:
		return x.ConnectionState.String()
	}
}
