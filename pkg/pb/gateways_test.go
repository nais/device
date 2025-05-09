package pb_test

import (
	"testing"

	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
)

func TestMergeGatewayHealth(t *testing.T) {
	newGateways := []*pb.Gateway{
		{
			Name:    "gw-1",
			Healthy: false,
		},
		{
			Name:    "gw-2",
			Healthy: false,
		},
	}

	existingGateways := []*pb.Gateway{
		{
			Name:    "gw-1",
			Healthy: true,
		},
		{
			Name:    "gw-2",
			Healthy: false,
		},
		{
			Name:    "gw-3",
			Healthy: true,
		},
	}

	updatedGateways := pb.MergeGatewayHealth(existingGateways, newGateways)

	assert.Equal(t, "gw-1", updatedGateways[0].Name)
	assert.True(t, updatedGateways[0].Healthy)

	assert.Equal(t, "gw-2", updatedGateways[1].Name)
	assert.False(t, updatedGateways[1].Healthy)

	assert.Lenf(t, updatedGateways, 2, "gw-3 should be removed as it's not in the newGateways list")
}
