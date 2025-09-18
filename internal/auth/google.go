package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

type Google struct {
	ClientID       string
	AllowedDomains []string
	jwks           jwk.CachedSet
}

const (
	googleJwksEndpoint = "https://www.googleapis.com/oauth2/v3/certs"
	googleIssuer       = "https://accounts.google.com"
)

func (g *Google) SetupJwkSetAutoRefresh(ctx context.Context) error {
	c, err := jwk.NewCache(
		ctx,
		httprc.NewClient(),
	)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}

	if err := c.Register(
		ctx,
		googleJwksEndpoint,
		jwk.WithMaxInterval(24*time.Hour*7),
		jwk.WithMinInterval(15*time.Minute),
	); err != nil {
		return fmt.Errorf("failed to register google JWKS: %w", err)
	}

	cs, err := c.CachedSet(googleJwksEndpoint)
	if err != nil {
		return fmt.Errorf("failed to get cached keyset: %w", err)
	}

	g.jwks = cs
	return nil
}

func (g *Google) JwtOptions() []jwt.ParseOption {
	return []jwt.ParseOption{
		jwt.WithValidate(true),
		jwt.WithKeySet(g.jwks, jws.WithInferAlgorithmFromKey(true)),
		jwt.WithAcceptableSkew(5 * time.Second),
		jwt.WithIssuer(googleIssuer),
		jwt.WithAudience(g.ClientID),
		jwt.WithRequiredClaim("hd"),
		jwt.WithRequiredClaim("email"),
	}
}

func (g *Google) TokenValidatorMiddleware() TokenValidator {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			bearer := strings.TrimSpace(r.Header.Get("Authorization"))
			token := strings.TrimSpace(strings.TrimPrefix(bearer, "Bearer"))

			user, err := g.ParseAndValidateToken(token)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			r = r.WithContext(WithEmail(r.Context(), user.Email))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func (g *Google) ParseAndValidateToken(token string) (*User, error) {
	tok, err := jwt.ParseString(token, g.JwtOptions()...)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	email, err := StringClaim("email", tok)
	if err != nil {
		return nil, fmt.Errorf("missing email claim: %w", err)
	}

	sub, _ := tok.Subject()
	if sub == "" {
		return nil, fmt.Errorf("empty sub claim in token")
	}

	hd, err := StringClaim("hd", tok)
	if err != nil {
		return nil, fmt.Errorf("missing hd claim: %w", err)
	}

	found := false
	for _, allowedDomain := range g.AllowedDomains {
		if hd == allowedDomain {
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("%q not in allowed domains: %v", hd, g.AllowedDomains)
	}

	return &User{
		ID:     sub,
		Email:  email,
		Groups: []string{"allUsers"},
	}, nil
}
