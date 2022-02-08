package wireguard_test

import (
	"bytes"
	"testing"

	"github.com/nais/device/pkg/device-agent/wireguard"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
)

func TestWGGenKey(t *testing.T) {
	privateKey := wireguard.WgGenKey()
	assert.Len(t, privateKey, 32)
	privateKeyB64 := wireguard.KeyToBase64(privateKey)
	assert.Len(t, privateKeyB64, 44)
}

func TestMarshalGateway(t *testing.T) {
	gw := &pb.Gateway{
		PublicKey: "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		Endpoint:  "13.37.13.37:51820",
		Ip:        "10.255.240.2/32",
		Routes:    []string{"13.37.69.0/24", "13.37.59.69/32"},
	}

	buf := new(bytes.Buffer)
	err := gw.WritePeerConfig(buf)

	assert.NoError(t, err)

	expected := `[Peer]
PublicKey = PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=
AllowedIPs = 13.37.69.0/24,13.37.59.69/32,10.255.240.2/32
Endpoint = 13.37.13.37:51820

`
	assert.Equal(t, expected, buf.String())
}
