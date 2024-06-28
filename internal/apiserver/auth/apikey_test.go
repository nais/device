package auth_test

import (
	"context"
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

	ctx := context.Background()

	assert.NoError(t, authenticator.Authenticate(ctx, "foo", "bar"))
	assert.NoError(t, authenticator.Authenticate(ctx, "bar", "baz"))

	errMsg := "invalid username or password"

	assert.Errorf(t, authenticator.Authenticate(ctx, "foo", "bloat"), errMsg)
	assert.Errorf(t, authenticator.Authenticate(ctx, "foo", ""), errMsg)
	assert.Errorf(t, authenticator.Authenticate(ctx, "", ""), errMsg)
	assert.Errorf(t, authenticator.Authenticate(ctx, "", "bar"), errMsg)
	assert.Errorf(t, authenticator.Authenticate(ctx, "bar", "bar"), errMsg)
	assert.Errorf(t, authenticator.Authenticate(ctx, "baz", "bar"), errMsg)
}
