package bootstrap_api

import (
	"fmt"
	"net/http"
	"strings"
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

func TokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(TokenHeaderKey)

		if !gatewayEnrollments.hasToken(token) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
