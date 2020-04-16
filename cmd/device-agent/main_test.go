package main_test

import (
	"testing"

	main "github.com/nais/device/cmd/device-agent"
	"github.com/stretchr/testify/assert"
)

func TestParseBootstrapToken(t *testing.T) {
	/*
		{
		  "deviceIP": "10.1.1.1",
		  "publicKey": "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		  "endpoint": "69.1.1.1:51820",
		  "apiServerIP": "10.1.1.2"
		}
	*/
	bootstrapToken := "ewogICJkZXZpY2VJUCI6ICIxMC4xLjEuMSIsCiAgInB1YmxpY0tleSI6ICJQUUttcmFQT1B5ZTVDSnExeDduanBsOHJSdTVSU3JJS3lIdlpYdEx2UzBFPSIsCiAgImVuZHBvaW50IjogIjY5LjEuMS4xOjUxODIwIiwKICAiYXBpU2VydmVySVAiOiAiMTAuMS4xLjIiCn0K"
	bootstrapConfig, err := main.ParseBootstrapToken(bootstrapToken)
	assert.NoError(t, err)
	assert.Equal(t, "10.1.1.1", bootstrapConfig.DeviceIP)
	assert.Equal(t, "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=", bootstrapConfig.PublicKey)
	assert.Equal(t, "69.1.1.1:51820", bootstrapConfig.Endpoint)
	assert.Equal(t, "10.1.1.2", bootstrapConfig.APIServerIP)
}

func TestGenerateWGConfig(t *testing.T) {
	bootstrapConfig := &main.BootstrapConfig{
		DeviceIP:    "10.1.1.1",
		PublicKey:   "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		Endpoint:    "69.1.1.1:51820",
		APIServerIP: "10.1.1.2",
	}
	privateKey := []byte("wFTAVe1stJPp0xQ+FE9so56uKh0jaHkPxJ4d2x9jPmU=")
	wgConfig := main.GenerateWireGuardConfig(bootstrapConfig, privateKey)

	expected := `
[Interface]
PrivateKey = wFTAVe1stJPp0xQ+FE9so56uKh0jaHkPxJ4d2x9jPmU=

[Peer]
PublicKey = PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=
AllowedIPs = 10.1.1.2
TunnelEndpoint = 69.1.1.1:51820
`
	assert.Equal(t, expected, string(wgConfig))

}
