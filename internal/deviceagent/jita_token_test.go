package deviceagent

import (
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwt"

	"github.com/nais/device/internal/deviceagent/auth"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestLoginIDTokenAsJita(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	idToken := func(t *testing.T, expiry time.Time) string {
		t.Helper()
		tok := jwt.New()
		require.NoError(t, tok.Set(jwt.ExpirationKey, expiry))
		signed, err := jwt.Sign(tok, jwa.HS256, []byte("secret"))
		require.NoError(t, err)
		return string(signed)
	}

	t.Run("valid token is reused", func(t *testing.T) {
		expiry := now.Add(30 * time.Minute)
		raw := idToken(t, expiry)

		got := loginIDTokenAsJita(raw, now)

		require.NotNil(t, got)
		assert.Equal(t, raw, got.AccessToken)
		assert.True(t, expiry.Equal(got.Expiry))
		assert.True(t, got.Valid())
	})

	t.Run("token about to expire is not reused", func(t *testing.T) {
		assert.Nil(t, loginIDTokenAsJita(idToken(t, now.Add(time.Minute)), now))
	})

	t.Run("expired token is not reused", func(t *testing.T) {
		assert.Nil(t, loginIDTokenAsJita(idToken(t, now.Add(-time.Hour)), now))
	})

	t.Run("token without expiry is not reused", func(t *testing.T) {
		signed, err := jwt.Sign(jwt.New(), jwa.HS256, []byte("secret"))
		require.NoError(t, err)
		assert.Nil(t, loginIDTokenAsJita(string(signed), now))
	})

	t.Run("garbage and empty tokens are not reused", func(t *testing.T) {
		assert.Nil(t, loginIDTokenAsJita("not-a-jwt", now))
		assert.Nil(t, loginIDTokenAsJita("", now))
	})
}
