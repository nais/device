package bootstrap_api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

func Credentials(credentialEntries []string) (map[string]string, error) {
	credentials := make(map[string]string)
	for _, key := range credentialEntries {
		entry := strings.Split(key, ":")
		if len(entry) > 2 {
			return nil, fmt.Errorf("invalid format on credentials, should be comma-separated entries on format 'user:key'")
		}

		credentials[entry[0]] = entry[1]
	}

	return credentials, nil
}

func (api *GatewayApi) gatewayAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gatewayName, token, ok := r.BasicAuth()

		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			log.Warnf("Unauthorized: no basic auth provided")
			return
		}

		if !api.authenticated(gatewayName, token) {
			w.WriteHeader(http.StatusUnauthorized)
			log.Warnf("Unauthorized: invalid credentials")
			return
		}

		ctx := context.WithValue(r.Context(), GatewayNameContextKey, gatewayName)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
