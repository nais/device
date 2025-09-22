package api

import (
	"slices"

	"github.com/nais/device/internal/apiserver/jita"
	"github.com/nais/device/pkg/pb"
)

const GatewayMSLoginName = "nais-device-gw-ms-login"

func filterList[T any](elements []T, filters ...func(T) bool) []T {
	var filtered []T
	for _, element := range elements {
		if allFiltersMatch(element, filters) {
			filtered = append(filtered, element)
		}
	}
	return filtered
}

func allFiltersMatch[T any](element T, filters []func(T) bool) bool {
	for _, filter := range filters {
		if !filter(element) {
			return false
		}
	}
	return true
}

func slicesHasIntersect[T comparable](sliceA []T, sliceB []T) bool {
	for _, a := range sliceA {
		if slices.Contains(sliceB, a) {
			return true
		}
	}
	return false
}

func not[T comparable](f func(T) bool) func(T) bool {
	return func(v T) bool {
		return !f(v)
	}
}

// ---
// Session filters
// ---
func sessionUserHasApproved(approvedUsers map[string]struct{}) func(session *pb.Session) bool {
	return func(session *pb.Session) bool {
		_, approved := approvedUsers[session.ObjectID]
		return approved
	}
}

func sessionIsHealthy(session *pb.Session) bool {
	return session.GetDevice().Healthy()
}

func sessionForGatewayGroups(gatewayGroups []string) func(*pb.Session) bool {
	return func(session *pb.Session) bool {
		return slicesHasIntersect(session.Groups, gatewayGroups)
	}
}

func sessionIsPrivileged(privilegedUsers []jita.PrivilegedUser) func(*pb.Session) bool {
	return func(session *pb.Session) bool {
		for _, privilegedUser := range privilegedUsers {
			if privilegedUser.UserId == session.ObjectID {
				return true
			}
		}
		return false
	}
}

func sessionIsPlatform(platform string) func(*pb.Session) bool {
	return func(session *pb.Session) bool {
		return session.GetDevice().GetPlatform() == platform
	}
}

// ---
// Gateway filters
// ---
func gatewayHasName(name string) func(*pb.Gateway) bool {
	return func(gateway *pb.Gateway) bool {
		return gateway.Name == name
	}
}

func gatewayForUserGroups(userGroups []string) func(*pb.Gateway) bool {
	return func(gateway *pb.Gateway) bool {
		return slicesHasIntersect(gateway.AccessGroupIDs, userGroups)
	}
}
