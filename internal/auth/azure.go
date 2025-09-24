package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
)

type Azure struct {
	ClientID       string
	Tenant         string
	jwkAutoRefresh *jwk.AutoRefresh
	ctx            context.Context
}

func (a Azure) JwksEndpoint() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/keys", a.Tenant)
}

func (a Azure) Issuer() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", a.Tenant)
}

func (a *Azure) SetupJwkSetAutoRefresh(ctx context.Context) error {
	ar := jwk.NewAutoRefresh(ctx)
	ar.Configure(a.JwksEndpoint(), jwk.WithMinRefreshInterval(time.Hour))

	// trigger initial token fetch
	_, err := ar.Refresh(ctx, a.JwksEndpoint())
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}

	a.ctx = ctx
	a.jwkAutoRefresh = ar
	return nil
}

func (a *Azure) KeySetFrom(_ jwt.Token) (jwk.Set, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	return a.jwkAutoRefresh.Fetch(ctx, a.JwksEndpoint())
}

func (a *Azure) JwtOptions() []jwt.ParseOption {
	return []jwt.ParseOption{
		jwt.WithValidate(true),
		jwt.InferAlgorithmFromKey(true),
		jwt.WithKeySetProvider(a),
		jwt.WithAcceptableSkew(5 * time.Second),
		jwt.WithIssuer(a.Issuer()),
		jwt.WithAudience(a.ClientID),
	}
}

func (a *Azure) TokenValidatorMiddleware() TokenValidator {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token, err := jwt.ParseHeader(r.Header, "Authorization", a.JwtOptions()...)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims, err := token.AsMap(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			username := claims["preferred_username"].(string)
			r = r.WithContext(WithEmail(r.Context(), username))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
