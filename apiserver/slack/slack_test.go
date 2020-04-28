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
		  "accessToken": "access_token"
		}
	*/
	enrollmentToken, err := slack.ParseEnrollmentToken("eyJzZXJpYWwiOiJzZXJpYWwiLCJwdWJsaWNLZXkiOiJwdWJsaWNfa2V5IiwiYWNjZXNzVG9rZW4iOiJhY2Nlc3NfdG9rZW4ifQ==")
	assert.NoError(t, err)
	assert.Equal(t, "serial", enrollmentToken.Serial)
	assert.Equal(t, "public_key", enrollmentToken.PublicKey)
	assert.Equal(t, "access_token", enrollmentToken.AccessToken)
}
