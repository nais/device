package token

import "net/http"

type (
	Validator func(http.Handler) http.Handler
	Parser    interface {
		ParseString(string) (*User, error)
		ParseHeader(http.Header, string) (*User, error)
	}
)
