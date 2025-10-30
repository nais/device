package auth

import (
	"context"
	"net/http"
)

type (
	TokenValidator func(http.Handler) http.Handler
	contextKey     string

	TokenParser interface {
		ParseString(string) (*User, error)
		ParseHeader(http.Header, string) (*User, error)
	}
	User struct {
		ID     string
		Email  string
		Groups []string
	}
)

const contextKeyEmail contextKey = "email"

func GetEmail(ctx context.Context) string {
	email, _ := ctx.Value(contextKeyEmail).(string)
	return email
}

func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, contextKeyEmail, email)
}
