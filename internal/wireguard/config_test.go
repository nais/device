package wireguard_test

import (
	"bytes"
	"testing"

	"github.com/nais/device/internal/wireguard"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
)

func TestConfig_MarshalINI(t *testing.T) {
	cfg := &wireguard.Config{
		AddressV4:  "address",
		ListenPort: 12345,
		MTU:        123,
		Peers: []wireguard.Peer{
			&pb.Gateway{
				Name:       "gw-name",
				PublicKey:  "gw-pubkey",
				Endpoint:   "gw-ep",
				Ipv4:       "gw-ip",
				Ipv6:       "gw-ipv6",
				RoutesIPv4: []string{"route1/32", "route2/24"},
			},
			&pb.Device{
				Serial:    "device-serial",
				PublicKey: "device-pubkey",
				Ipv4:      "device-private-ip",
				Ipv6:      "device-private-ipv6",
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
AllowedIPs = route1/32,route2/24,gw-ip/32,gw-ipv6/128
Endpoint = gw-ep

[Peer] # device-serial
PublicKey = device-pubkey
AllowedIPs = device-private-ip/32,device-private-ipv6/128

`

	assert.Equal(t, expected, buf.String())
}

func TestConfig_MarshalINI_Minimal(t *testing.T) {
	cfg := &wireguard.Config{
		PrivateKey: "privkey",
		Peers: []wireguard.Peer{
			&pb.Gateway{
				Name:       "gw-name",
				PublicKey:  "gw-pubkey",
				Endpoint:   "gw-ep",
				Ipv4:       "gw-ip",
				RoutesIPv4: []string{},
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
