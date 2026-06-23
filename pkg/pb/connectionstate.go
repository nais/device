package pb

// see time.Format for format documentation, equals %H:%M:%S
const timeFormat = "15:04:05"

// IssueTitleDosDontsNotAccepted is the canonical title for the Do's and don'ts acceptance issue.
const IssueTitleDosDontsNotAccepted = "Do's and don'ts not accepted"

// ConnectionStateString returns a  human-friendly connection status
func (x *AgentStatus) ConnectionStateString() string {
	switch x.ConnectionState {
	case AgentState_Bootstrapping:
		return "Bootstrapping device"
	case AgentState_Unhealthy:
		for _, issue := range x.Issues {
			if issue.GetTitle() == IssueTitleDosDontsNotAccepted {
				return "No access: accept Do's and don'ts"
			}
		}

		return "No access, check Kolide tray icon"
	case AgentState_Disconnecting:
		return "Disconnecting"
	case AgentState_Authenticating:
		return "Authenticating"
	case AgentState_AuthenticateBackoff:
		return "Authentication failed; waiting to retry"
	case AgentState_Connected:
		return "Connected since " + x.ConnectedSince.AsTime().Local().Format(timeFormat)
	default:
		return x.ConnectionState.String()
	}
}
