package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
)

type Google struct {
	ClientID       string
	AllowedDomains []string
	jwkAutoRefresh *jwk.AutoRefresh
	ctx            context.Context
}

const (
	googleJwksEndpoint = "https://www.googleapis.com/oauth2/v3/certs"
	googleIssuer       = "https://accounts.google.com"
)

func (g *Google) SetupJwkSetAutoRefresh(ctx context.Context) error {
	ar := jwk.NewAutoRefresh(ctx)
	ar.Configure(googleJwksEndpoint, jwk.WithMinRefreshInterval(time.Hour))

	// trigger initial token fetch
	_, err := ar.Refresh(ctx, googleJwksEndpoint)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}

	g.ctx = ctx
	g.jwkAutoRefresh = ar
	return nil
}

func (g *Google) KeySetFrom(t jwt.Token) (jwk.Set, error) {
	ctx, cancel := context.WithTimeout(g.ctx, 10*time.Second)
	defer cancel()

	return g.jwkAutoRefresh.Fetch(ctx, googleJwksEndpoint)
}

func (g *Google) JwtOptions() []jwt.ParseOption {
	return []jwt.ParseOption{
		jwt.WithValidate(true),
		jwt.InferAlgorithmFromKey(true),
		jwt.WithAcceptableSkew(5 * time.Second),
		jwt.WithIssuer(googleIssuer),
		jwt.WithKeySetProvider(g),
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

	emailClaim, _ := tok.Get("email")
	email, _ := emailClaim.(string)
	if email == "" {
		return nil, fmt.Errorf("empty email claim in token")
	}

	subClaim, _ := tok.Get("sub")
	sub, _ := subClaim.(string)
	if sub == "" {
		return nil, fmt.Errorf("empty sub claim in token")
	}

	hdClaim, _ := tok.Get("hd")
	hd, _ := hdClaim.(string)

	found := false
	for _, allowedDomain := range g.AllowedDomains {
		if hd == allowedDomain {
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("'%s' not in allowed domains: %v", hd, g.AllowedDomains)
	}

	return &User{
		ID:     sub,
		Email:  email,
		Groups: []string{"allUsers"},
	}, nil
}
