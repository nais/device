package slack_test

import (
	"testing"

	"github.com/nais/device/apiserver/slack"
	"github.com/stretchr/testify/assert"
)

func TestParseEnrollmentToken(t *testing.T) {
	/*
		{
		  "serial": "serial",
		  "publicKey": "public_key",
		  "platform": "platform"
		}
	*/
	enrollmentToken, err := slack.ParseEnrollmentToken("ewogICJzZXJpYWwiOiAic2VyaWFsIiwKICAicHVibGljS2V5IjogInB1YmxpY19rZXkiLAogICJwbGF0Zm9ybSI6ICJwbGF0Zm9ybSIKfQo=\n")
	assert.NoError(t, err)
	assert.Equal(t, "serial", enrollmentToken.Serial)
	assert.Equal(t, "public_key", enrollmentToken.PublicKey)
	assert.Equal(t, "platform", enrollmentToken.Platform)
}
