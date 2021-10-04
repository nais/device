package azure

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
	log "github.com/sirupsen/logrus"
)

const NaisDeviceApprovalGroup = "ffd89425-c75c-4618-b5ab-67149ddbbc2d"

type Azure struct {
	ClientID     string
	ClientSecret string
	Jwks         jwk.Set
	Tenant       string
}

func (a Azure) DiscoveryURL() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/keys", a.Tenant)
}
func (a Azure) Issuer() string {
	return fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", a.Tenant)
}

func (a *Azure) FetchCertificates() error {
	ctx := context.Background()
	jwks, err := jwk.Fetch(ctx, a.DiscoveryURL())

	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}

	a.Jwks = jwks
	return nil
}

func failAuth(w http.ResponseWriter, cause error) {
	log.Warn(cause)
	w.WriteHeader(http.StatusForbidden)
	_, err := fmt.Fprintf(w, "Unauthorized: %s", cause)
	if err != nil {
		log.Errorf("Writing http response: %v", err)
	}
}

func (a *Azure) JwtOptions() []jwt.ParseOption {
	return []jwt.ParseOption{
		jwt.WithValidate(true),
		jwt.InferAlgorithmFromKey(true),
		jwt.WithAcceptableSkew(5 * time.Second),
		jwt.WithIssuer(a.Issuer()),
		jwt.WithKeySet(a.Jwks),
		jwt.WithAudience(a.ClientID),
	}
}

func (a *Azure) TokenValidatorMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			token, err := jwt.ParseHeader(r.Header, "Authorization", a.JwtOptions()...)

			if err != nil {
				failAuth(w, fmt.Errorf("parse token: %w", err))
				return
			}

			claims, err := token.AsMap(r.Context())
			if err != nil {
				failAuth(w, fmt.Errorf("convert claims to map: %s", err))
				return
			}

			username := claims["preferred_username"].(string)

			if !UserInNaisdeviceApprovalGroup(claims) {
				w.WriteHeader(http.StatusSeeOther)
				http.Redirect(w, r, "https://naisdevice-approval.nais.io/", http.StatusSeeOther)
				log.Infof("Redirected user(%s) to do's and don'ts", username)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), "preferred_username", username))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func UserInNaisdeviceApprovalGroup(claims map[string]interface{}) bool {
	for _, group := range claims["groups"].([]interface{}) {
		if group.(string) == NaisDeviceApprovalGroup {
			return true
		}
	}

	return false
}
