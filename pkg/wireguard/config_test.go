package wireguard_test

import (
	"bytes"
	"testing"

	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/wireguard"
	"github.com/stretchr/testify/assert"
)

func TestConfig_MarshalINI(t *testing.T) {
	cfg := &wireguard.Config{
		Address:    "address",
		ListenPort: 12345,
		MTU:        123,
		Peers: []wireguard.Peer{
			&pb.Gateway{
				Name:      "gw-name",
				PublicKey: "gw-pubkey",
				Endpoint:  "gw-ep",
				Ipv4:      "gw-ip",
				Routes:    []string{"route1/32", "route2/24"},
			},
			&pb.Device{
				Serial:    "device-serial",
				PublicKey: "device-pubkey",
				Ipv4:      "device-private-ip",
			},
		},
		PrivateKey: "privkey",
	}

	buf := &bytes.Buffer{}
	err := cfg.MarshalINI(buf)

	assert.NoError(t, err)

	expected := `[Interface]
PrivateKey = privkey
ListenPort = 12345
MTU = 123
Address = address

[Peer] # gw-name
PublicKey = gw-pubkey
AllowedIPs = route1/32,route2/24,gw-ip/32
Endpoint = gw-ep

[Peer] # device-serial
PublicKey = device-pubkey
AllowedIPs = device-private-ip/32

`

	assert.Equal(t, expected, buf.String())
}

func TestConfig_MarshalINI_Minimal(t *testing.T) {
	cfg := &wireguard.Config{
		PrivateKey: "privkey",
		Peers: []wireguard.Peer{
			&pb.Gateway{
				Name:      "gw-name",
				PublicKey: "gw-pubkey",
				Endpoint:  "gw-ep",
				Ipv4:      "gw-ip",
				Routes:    []string{},
			},
		},
	}

	buf := &bytes.Buffer{}
	err := cfg.MarshalINI(buf)

	assert.NoError(t, err)

	expected := `[Interface]
PrivateKey = privkey

[Peer] # gw-name
PublicKey = gw-pubkey
AllowedIPs = gw-ip/32
Endpoint = gw-ep

`

	assert.Equal(t, expected, buf.String())
}
