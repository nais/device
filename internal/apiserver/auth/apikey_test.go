package auth_test

import (
	"testing"

	"github.com/nais/device/internal/apiserver/auth"
	"github.com/stretchr/testify/assert"
)

func TestApikeyAuthenticator_Authenticate(t *testing.T) {
	users := map[string]string{
		"foo": "bar",
		"bar": "baz",
	}
	authenticator := auth.NewAPIKeyAuthenticator(users)

	assert.NoError(t, authenticator.Authenticate("foo", "bar"))
	assert.NoError(t, authenticator.Authenticate("bar", "baz"))

	errMsg := "invalid username or password"

	assert.Errorf(t, authenticator.Authenticate("foo", "bloat"), errMsg)
	assert.Errorf(t, authenticator.Authenticate("foo", ""), errMsg)
	assert.Errorf(t, authenticator.Authenticate("", ""), errMsg)
	assert.Errorf(t, authenticator.Authenticate("", "bar"), errMsg)
	assert.Errorf(t, authenticator.Authenticate("bar", "bar"), errMsg)
	assert.Errorf(t, authenticator.Authenticate("baz", "bar"), errMsg)
}
