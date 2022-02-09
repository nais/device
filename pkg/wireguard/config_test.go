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
		PrivateKey: "privkey",
		Interface:  "utun66",
		ListenPort: 12345,
		Peers: []wireguard.Peer{
			&pb.Gateway{
				Name:      "gw-name",
				PublicKey: "gw-pubkey",
				Endpoint:  "gw-ep",
				Ip:        "gw-ip",
				Routes:    []string{"route1", "route2"},
			},
			&pb.Device{
				Serial:    "device-serial",
				PublicKey: "device-pubkey",
				Ip:        "device-private-ip",
			},
		},
	}

	buf := &bytes.Buffer{}
	err := cfg.MarshalINI(buf)

	assert.NoError(t, err)

	expected := `[Interface]
PrivateKey = privkey
ListenPort = 12345
Interface = utun66

[Peer] # gw-name
PublicKey = gw-pubkey
AllowedIPs = route1,route2,gw-ip
Endpoint = gw-ep

[Peer] # device-serial
PublicKey = device-pubkey
AllowedIPs = device-private-ip

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
				Ip:        "gw-ip",
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
AllowedIPs = gw-ip
Endpoint = gw-ep

`

	assert.Equal(t, expected, buf.String())
}
