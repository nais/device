package wireguard

import (
	"testing"

	"github.com/nais/device/device-agent/apiserver"
	"github.com/stretchr/testify/assert"
)

func TestWGGenKey(t *testing.T) {
	privateKey := WgGenKey()
	assert.Len(t, privateKey, 32)
	privateKeyB64 := KeyToBase64(privateKey)
	assert.Len(t, privateKeyB64, 44)
}

func TestGenerateWireGuardPeers(t *testing.T) {
	gateway := apiserver.Gateway{
		PublicKey: "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		Endpoint:  "13.37.13.37:51820",
		IP:        "10.255.240.2",
		Healthy:   true,
		Routes:    []string{"13.37.69.0/24", "13.37.59.69/32"},
	}
	gateways := apiserver.Gateways{&gateway}

	config := gateways.MarshalIni()
	expected := `[Peer]
PublicKey = PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=
AllowedIPs = 10.255.240.2/32,13.37.69.0/24,13.37.59.69/32
Endpoint = 13.37.13.37:51820
`
	assert.Equal(t, expected, string(config))
}
