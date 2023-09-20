package pb_test

import (
	"testing"

	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
)

func TestMergeGatewayHealth(t *testing.T) {
	src := []*pb.Gateway{
		{
			Name:    "gw-1",
			Healthy: false,
		},
		{
			Name:    "gw-2",
			Healthy: true,
		},
	}

	dst := []*pb.Gateway{
		{
			Name:    "gw-1",
			Healthy: true,
			Ipv4:    "foo",
		},
		{
			Name:    "gw-2",
			Healthy: false,
			Ipv4:    "foo",
		},
		{
			Name:    "gw-3",
			Healthy: true,
			Ipv4:    "foo",
		},
		{
			Name:    "gw-4",
			Healthy: false,
			Ipv4:    "foo",
			Ipv6:    "ffff::1",
		},
	}

	pb.MergeGatewayHealth(dst, src)

	// assert all ips (and other fields) preserved
	for _, gw := range dst {
		assert.Equal(t, "foo", gw.Ipv4)
	}

	assert.Equal(t, "gw-1", dst[0].Name)
	assert.False(t, dst[0].Healthy)

	assert.Equal(t, "gw-2", dst[1].Name)
	assert.True(t, dst[1].Healthy)

	assert.Equal(t, "gw-3", dst[2].Name)
	assert.True(t, dst[2].Healthy)

	assert.Equal(t, "gw-4", dst[3].Name)
	assert.False(t, dst[3].Healthy)
}
