package main_test

import (
	"testing"

	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/wireguard"
	"github.com/stretchr/testify/assert"
)

func TestParseBootstrapToken(t *testing.T) {
	/*
		{
		  "deviceIP": "10.1.1.1",
		  "publicKey": "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		  "tunnelEndpoint": "69.1.1.1:51820",
		  "apiServerIP": "10.1.1.2"
		}
	*/
	bootstrapToken := "ewogICJkZXZpY2VJUCI6ICIxMC4xLjEuMSIsCiAgInB1YmxpY0tleSI6ICJQUUttcmFQT1B5ZTVDSnExeDduanBsOHJSdTVSU3JJS3lIdlpYdEx2UzBFPSIsCiAgInR1bm5lbEVuZHBvaW50IjogIjY5LjEuMS4xOjUxODIwIiwKICAiYXBpU2VydmVySVAiOiAiMTAuMS4xLjIiCn0K"
	bootstrapConfig, err := apiserver.ParseBootstrapToken(bootstrapToken)
	assert.NoError(t, err)
	assert.Equal(t, "10.1.1.1", bootstrapConfig.TunnelIP)
	assert.Equal(t, "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=", bootstrapConfig.PublicKey)
	assert.Equal(t, "69.1.1.1:51820", bootstrapConfig.Endpoint)
	assert.Equal(t, "10.1.1.2", bootstrapConfig.APIServerIP)
}

func TestGenerateWireGuardPeers(t *testing.T) {
	gateways := []apiserver.Gateway{{
		PublicKey: "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		Endpoint:  "13.37.13.37:51820",
		IP:        "10.255.240.2",
		Routes:    []string{"13.37.69.0/24", "13.37.59.69/32"},
	}}

	config := wireguard.GenerateWireGuardPeers(gateways)
	expected := `[Peer]
PublicKey = PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=
AllowedIPs = 13.37.69.0/24,13.37.59.69/32,10.255.240.2/32
Endpoint = 13.37.13.37:51820
`
	assert.Equal(t, expected, config)
}

func TestGenerateEnrollmentToken(t *testing.T) {
	expected := "eyJzZXJpYWwiOiJzZXJpYWwiLCJwdWJsaWNLZXkiOiJwdWJsaWNfa2V5IiwicGxhdGZvcm0iOiJwbGF0Zm9ybSJ9"
	enrollmentToken, err := apiserver.GenerateEnrollmentToken("serial", "platform", []byte("public_key"))

	assert.NoError(t, err)
	assert.Equal(t, expected, enrollmentToken, "interface changed, remember to change the apiserver counterpart")
}
