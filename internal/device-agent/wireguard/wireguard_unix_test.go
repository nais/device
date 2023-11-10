//go:build linux || darwin
// +build linux darwin

package wireguard_test

import (
	"bytes"
	"testing"

	"github.com/nais/device/internal/device-agent/wireguard"
	"github.com/nais/device/internal/pb"
	"github.com/stretchr/testify/assert"
)

func TestMarshalConfiguration(t *testing.T) {
	cfg := &pb.Configuration{
		PrivateKey: "abc",
		DeviceIPv4: "127.0.0.1",
		Gateways: []*pb.Gateway{
			{
				Name:       "gateway-1",
				PublicKey:  "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
				Endpoint:   "13.37.13.37:51820",
				Ipv4:       "10.255.240.2",
				Ipv6:       "fd00::2",
				RoutesIPv4: []string{"13.37.69.0/24", "13.37.59.69/32"},
			},
			{
				Name:       "gateway-2",
				PublicKey:  "foobar",
				Endpoint:   "14.37.13.37:51820",
				Ipv4:       "11.255.240.2",
				RoutesIPv4: []string{"14.37.69.0/24", "14.37.59.69/32"},
			},
		},
	}

	buf := new(bytes.Buffer)
	err := wireguard.Marshal(buf, cfg)

	assert.NoError(t, err)

	expected := `[Interface]
PrivateKey = abc

[Peer] # gateway-1
PublicKey = PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=
AllowedIPs = 13.37.69.0/24,13.37.59.69/32,10.255.240.2/32,fd00::2/128
Endpoint = 13.37.13.37:51820

[Peer] # gateway-2
PublicKey = foobar
AllowedIPs = 14.37.69.0/24,14.37.59.69/32,11.255.240.2/32
Endpoint = 14.37.13.37:51820

`
	assert.Equal(t, expected, buf.String())
}
