package auth

import (
	"net/http"
)

func MockTokenValidator() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w,
				r.WithContext(
					WithEmail(r.Context(), "username@mock.dev"),
				),
			)
		})
	}
}
