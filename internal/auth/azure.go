package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

const NaisDeviceApprovalGroup = "ffd89425-c75c-4618-b5ab-67149ddbbc2d"

type Azure struct {
	ClientID string
	Tenant   string
	jwks     jwk.CachedSet
	ctx      context.Context
}

func (a Azure) JwksEndpoint() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/keys", a.Tenant)
}

func (a Azure) Issuer() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", a.Tenant)
}

func (a *Azure) SetupJwkCache(ctx context.Context) error {
	c, err := jwk.NewCache(
		ctx,
		httprc.NewClient(),
	)
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}

	if err := c.Register(
		ctx,
		a.JwksEndpoint(),
		jwk.WithMaxInterval(24*time.Hour*7),
		jwk.WithMinInterval(15*time.Minute),
	); err != nil {
		return fmt.Errorf("failed to register google JWKS: %w", err)
	}

	cs, err := c.CachedSet(a.JwksEndpoint())
	if err != nil {
		return fmt.Errorf("failed to get cached keyset: %w", err)
	}

	a.jwks = cs
	return nil
}

func (a *Azure) JwtOptions() []jwt.ParseOption {
	return []jwt.ParseOption{
		jwt.WithValidate(true),
		jwt.WithKeySet(a.jwks, jws.WithInferAlgorithmFromKey(true)),
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

			groups, err := GroupsClaim(token)
			if err != nil {
				http.Error(w, "missing groups claim", http.StatusUnauthorized)
				return
			}

			if !UserInNaisdeviceApprovalGroup(groups) {
				w.WriteHeader(http.StatusSeeOther)
				http.Redirect(w, r, "https://naisdevice-approval.external.prod-gcp.nav.cloud.nais.io/", http.StatusSeeOther)
				return
			}

			username, err := StringClaim("preferred_username", token)
			if err != nil {
				http.Error(w, "missing preferred_username claim", http.StatusUnauthorized)
				return
			}
			r = r.WithContext(WithEmail(r.Context(), username))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func UserInNaisdeviceApprovalGroup(groups []string) bool {
	for _, group := range groups {
		if group == NaisDeviceApprovalGroup {
			return true
		}
	}

	return false
}
