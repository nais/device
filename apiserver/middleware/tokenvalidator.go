package middleware

import (
	"context"
	"fmt"
	"github.com/nais/device/apiserver/azure/discovery"
	"github.com/nais/device/apiserver/azure/validate"
	log "github.com/sirupsen/logrus"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth"
)

func TokenValidatorMiddleware(certificates map[string]discovery.CertificateList, audience string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			var claims jwt.MapClaims

			token := jwtauth.TokenFromHeader(r)

			_, err := jwt.ParseWithClaims(token, &claims, validate.JWTValidator(certificates, audience))
			if err != nil {
				w.WriteHeader(http.StatusForbidden)
				_, err = fmt.Fprintf(w, "Unauthorized access: %s", err.Error())
				if err != nil {
					log.Error("Writing http response: %v", err)
				}
				return
			}

			var groups []string
			groupInterface := claims["groups"].([]interface{})
			groups = make([]string, len(groupInterface))
			for i, v := range groupInterface {
				groups[i] = v.(string)
			}
			r = r.WithContext(context.WithValue(r.Context(), "groups", groups))
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

