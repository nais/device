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
		  "tunnelEndpoint": "69.1.1.1:51820",
		  "apiServerIP": "10.1.1.2"
		}
	*/
	bootstrapToken := "ewogICJkZXZpY2VJUCI6ICIxMC4xLjEuMSIsCiAgInB1YmxpY0tleSI6ICJQUUttcmFQT1B5ZTVDSnExeDduanBsOHJSdTVSU3JJS3lIdlpYdEx2UzBFPSIsCiAgInR1bm5lbEVuZHBvaW50IjogIjY5LjEuMS4xOjUxODIwIiwKICAiYXBpU2VydmVySVAiOiAiMTAuMS4xLjIiCn0K"
	bootstrapConfig, err := main.ParseBootstrapToken(bootstrapToken)
	assert.NoError(t, err)
	assert.Equal(t, "10.1.1.1", bootstrapConfig.TunnelIP)
	assert.Equal(t, "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=", bootstrapConfig.PublicKey)
	assert.Equal(t, "69.1.1.1:51820", bootstrapConfig.Endpoint)
	assert.Equal(t, "10.1.1.2", bootstrapConfig.APIServerIP)
}

func TestGenerateWireGuardPeers(t *testing.T) {
	gateways := []main.Gateway{{
		PublicKey: "PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=",
		Endpoint:  "13.37.13.37:51820",
		IP:        "10.255.240.2",
		Routes:    []string{"13.37.69.0/24", "13.37.59.69/32"},
	}}

	config := main.GenerateWireGuardPeers(gateways)
	expected := `[Peer]
PublicKey = PQKmraPOPye5CJq1x7njpl8rRu5RSrIKyHvZXtLvS0E=
AllowedIPs = 13.37.69.0/24,13.37.59.69/32,10.255.240.2/32
Endpoint = 13.37.13.37:51820
`
	assert.Equal(t, expected, config)
}

func TestGenerateEnrollmentToken(t *testing.T) {
	expected := "eyJzZXJpYWwiOiJzZXJpYWwiLCJwdWJsaWNLZXkiOiJwdWJsaWNfa2V5IiwiYWNjZXNzVG9rZW4iOiJhY2Nlc3NfdG9rZW4ifQ=="
	enrollmentToken, err := main.GenerateEnrollmentToken("serial", "access_token", []byte("public_key"))

	assert.NoError(t, err)
	assert.Equal(t, expected, enrollmentToken, "interface changed, remember to change the apiserver counterpart")
}

func TestWGGenKey(t *testing.T) {
	privateKey := main.WgGenKey()
	assert.Len(t, privateKey, 32)
	privateKeyB64 := main.KeyToBase64(privateKey)
	assert.Len(t, privateKeyB64, 44)
}
