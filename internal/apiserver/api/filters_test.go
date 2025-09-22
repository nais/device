package api

import (
	"testing"

	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
)

func TestRemoveMSGateway(t *testing.T) {
	gateways := []*pb.Gateway{
		{
			Name: "other",
		},
		{
			Name: GatewayMSLoginName,
		},
	}

	noMSGateway := filterList(gateways, not(gatewayHasName(GatewayMSLoginName)))
	unchanged := filterList(gateways, not(gatewayHasName(GatewayMSLoginName)))
	removeNonExistant := filterList(gateways, not(gatewayHasName("not-exists")))

	assert.Len(t, gateways, 2)

	assert.Len(t, noMSGateway, 1)
	assert.Equal(t, noMSGateway[0].Name, "other")
	assert.Len(t, unchanged, 1)
	assert.Equal(t, unchanged[0].Name, "other")

	assert.Len(t, removeNonExistant, 2)
}

func Test_sessionUserHasApproved(t *testing.T) {
	tests := []struct {
		name          string
		approvedUsers map[string]struct{}
		session       *pb.Session
		result        bool
	}{
		{
			name:          "has approved",
			approvedUsers: map[string]struct{}{"approvedUser": {}},
			session:       &pb.Session{ObjectID: "approvedUser"},
			result:        true,
		},
		{
			name:          "has not approved",
			approvedUsers: map[string]struct{}{"anotherUser": {}},
			session:       &pb.Session{ObjectID: "approvedUser"},
			result:        false,
		},
		{
			name:          "empty approved users",
			approvedUsers: nil,
			session:       &pb.Session{ObjectID: "approvedUser"},
			result:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.result, sessionUserHasApproved(tt.approvedUsers)(tt.session))
		})
	}
}
