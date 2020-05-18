package wireguard_test

import (
	"testing"

	"github.com/nais/device/device-agent/bootstrapper"
	"github.com/nais/device/device-agent/wireguard"
	"github.com/stretchr/testify/assert"
)

func TestGenerateWGConfig(t *testing.T) {
	bootstrapConfig := &bootstrapper.BootstrapConfig{
		DeviceIP:    "10.1.1.1",
		PublicKey:   "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		Endpoint:    "69.1.1.1:51820",
		APIServerIP: "10.1.1.2",
	}
	privateKey := []byte("wFTAVe1stJPp0xQ+FE9so56uKh0jaHkPxJ4d2x9jPmU=")
	wgConfig := wireguard.GenerateBaseConfig(bootstrapConfig, privateKey)

	expected := `[Interface]
PrivateKey = wFTAVe1stJPp0xQ+FE9so56uKh0jaHkPxJ4d2x9jPmU=
MTU = 1360
Address = 10.1.1.1

[Peer]
PublicKey = PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=
AllowedIPs = 10.1.1.2/32
Endpoint = 69.1.1.1:51820
`
	assert.Equal(t, expected, wgConfig)
}
