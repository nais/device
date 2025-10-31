package token

import (
	"net/http"
)

func Middleware(h Parser) Validator {
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

func MockMiddleware() func(next http.Handler) http.Handler {
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
