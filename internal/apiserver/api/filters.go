package api

import (
	"time"

	"github.com/nais/device/internal/apiserver/jita"
	apiserver_metrics "github.com/nais/device/internal/apiserver/metrics"
	"github.com/nais/device/internal/pb"
)

func AfterGracePeriod(d *pb.DeviceIssue) bool {
	return time.Now().After(d.GetResolveBefore().AsTime())
}

// Return a list of user sessions that are authorized to access a gateway through JITA.
func privileged(jita jita.Client, gateway *pb.Gateway, sessions []*pb.Session) []*pb.Session {
	if !gateway.RequiresPrivilegedAccess {
		return sessions
	}
	privilegedUsers := jita.GetPrivilegedUsersForGateway(gateway.Name)

	m, _ := apiserver_metrics.PrivilegedUsersPerGateway.GetMetricWithLabelValues(gateway.Name)
	m.Set(float64(len(privilegedUsers)))

	var sessionsToReturn []*pb.Session
	for _, session := range sessions {
		if userIsPrivileged(privilegedUsers, session.ObjectID) {
			sessionsToReturn = append(sessionsToReturn, session)
		}
	}

	return sessionsToReturn
}

// Return all healthy devices in a set of devices.
// Healthy means that Kolide has reported the device as "active recently enough"
// and doesn't have any severe outstanding issues.
func healthy(devices []*pb.Device) []*pb.Device {
	var healthyDevices []*pb.Device

	for _, device := range devices {
		if device.GetHealthy() {
			healthyDevices = append(healthyDevices, device)
		}
	}

	return healthyDevices
}

// Find all sessions that are authorized to access a gateway,
// then return a list of all device configurations belonging to those sessions.
func authorized(gatewayGroups []string, sessions []*pb.Session) []*pb.Device {
	var authorizedDevices []*pb.Device

	for _, session := range sessions {
		if StringSliceHasIntersect(session.Groups, gatewayGroups) {
			authorizedDevices = append(authorizedDevices, session.Device)
		}
	}

	return authorizedDevices
}

// filter out duplicate devices (duplicate entries cause issues with the generated config on the gateway)
func unique(devices []*pb.Device) []*pb.Device {
	visited := make(map[int64]struct{})
	var ret []*pb.Device
	for _, d := range devices {
		if _, exists := visited[d.GetId()]; exists {
			continue
		}

		visited[d.GetId()] = struct{}{}
		ret = append(ret, d)
	}

	return ret
}

func StringSliceHasIntersect(slice1 []string, slice2 []string) bool {
	for _, a := range slice1 {
		for _, b := range slice2 {
			if a == b {
				return true
			}
		}
	}
	return false
}

func userIsPrivileged(privilegedUsers []jita.PrivilegedUser, users string) bool {
	for _, privilegedUser := range privilegedUsers {
		if privilegedUser.UserId == users {
			return true
		}
	}
	return false
}
