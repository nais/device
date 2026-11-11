package deviceagent

import (
	"testing"
	"time"

	"github.com/nais/device/internal/deviceagent/auth"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestJitaToken(t *testing.T) {
	expiry := time.Now().Add(time.Hour)
	tokens := &auth.Tokens{
		Token:   &oauth2.Token{AccessToken: "access-token", Expiry: expiry},
		IDToken: "id-token",
	}

	t.Run("azure sends the access token", func(t *testing.T) {
		got := jitaToken(pb.AuthProvider_Azure, tokens)
		assert.Equal(t, "access-token", got.AccessToken)
		assert.True(t, got.Valid())
	})

	t.Run("google sends the id token and keeps the expiry", func(t *testing.T) {
		got := jitaToken(pb.AuthProvider_Google, tokens)
		assert.Equal(t, "id-token", got.AccessToken)
		assert.Equal(t, expiry, got.Expiry)
		assert.True(t, got.Valid())
	})

	t.Run("google without id token is invalid", func(t *testing.T) {
		got := jitaToken(pb.AuthProvider_Google, &auth.Tokens{Token: tokens.Token})
		assert.False(t, got.Valid())
	})
}
