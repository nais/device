package api

import (
	"testing"

	"github.com/nais/device/internal/pb"
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
