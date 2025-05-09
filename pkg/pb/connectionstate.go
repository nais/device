package pb

// see time.Format for format documentation, equals %H:%M:%S
const timeFormat = "15:04:05"

// Human-friendly connection status
func (x *AgentStatus) ConnectionStateString() string {
	switch x.ConnectionState {
	case AgentState_Bootstrapping:
		return "Bootstrapping device..."
	case AgentState_Unhealthy:
		return "Device is unhealthy; no access to resources"
	case AgentState_Disconnecting:
		return "Disconnecting..."
	case AgentState_Authenticating:
		return "Authenticating..."
	case AgentState_AuthenticateBackoff:
		return "Authentication failed; waiting to retry..."
	case AgentState_Connected:
		return "Connected since " + x.ConnectedSince.AsTime().Format(timeFormat)
	default:
		return x.ConnectionState.String()
	}
}
