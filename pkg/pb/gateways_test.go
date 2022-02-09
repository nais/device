package pb_test

import (
	"bytes"
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
			Ip:      "foo",
		},
		{
			Name:    "gw-2",
			Healthy: false,
			Ip:      "foo",
		},
		{
			Name:    "gw-3",
			Healthy: true,
			Ip:      "foo",
		},
		{
			Name:    "gw-4",
			Healthy: false,
			Ip:      "foo",
		},
	}

	pb.MergeGatewayHealth(dst, src)

	// assert all ips (and other fields) preserved
	for _, gw := range dst {
		assert.Equal(t, "foo", gw.Ip)
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

func TestWriteGatewayPeerConfig(t *testing.T) {
	gateway := &pb.Gateway{
		PublicKey: "pub",
		Ip:        "ip",
		Endpoint:  "endpoint",
	}

	buf := new(bytes.Buffer)
	err := gateway.WritePeerConfig(buf)
	assert.NoError(t, err)

	expected := `[Peer]
PublicKey = pub
AllowedIPs = ip
Endpoint = endpoint

`
	assert.Equal(t, expected, buf.String())
}
