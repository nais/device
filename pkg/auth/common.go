package auth

import (
	"context"
	"net/http"
)

type User struct {
	ID     string
	Email  string
	Groups []string
}

type (
	TokenValidator func(http.Handler) http.Handler
	contextKey     string
)

const contextKeyEmail contextKey = "email"

func GetEmail(ctx context.Context) string {
	email, _ := ctx.Value(contextKeyEmail).(string)
	return email
}

func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, contextKeyEmail, email)
}
