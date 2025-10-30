package auth

import (
	"net/http"
)

func Middleware(h TokenParser) TokenValidator {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			user, err := h.ParseHeader(r.Header, "Authorization")
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
