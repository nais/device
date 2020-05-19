// +build linux darwin

package wireguard_test

import (
	"github.com/nais/device/pkg/bootstrap"
	"testing"

	"github.com/nais/device/device-agent/wireguard"
	"github.com/stretchr/testify/assert"
)

func TestGenerateWGConfig(t *testing.T) {
	bootstrapConfig := &bootstrap.Config{
		DeviceIP:       "10.1.1.1",
		PublicKey:      "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		TunnelEndpoint: "69.1.1.1:51820",
		APIServerIP:    "10.1.1.2",
	}
	privateKey := []byte("wFTAVe1stJPp0xQ+FE9so56uKh0jaHkPxJ4d2x9jPmU=")

	wgConfig := wireguard.GenerateBaseConfig(bootstrapConfig, privateKey)
	expected := `[Interface]
PrivateKey = d0ZUQVZlMXN0SlBwMHhRK0ZFOXNvNTZ1S2gwamFIa1B4SjRkMng5alBtVT0=

[Peer]
PublicKey = PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=
AllowedIPs = 10.1.1.2/32
Endpoint = 69.1.1.1:51820
`
	assert.Equal(t, expected, wgConfig)
}
