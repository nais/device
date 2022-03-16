package auth

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
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

func failAuth(w http.ResponseWriter, cause error) {
	log.Warn(cause)
	w.WriteHeader(http.StatusForbidden)
	_, err := fmt.Fprintf(w, "Unauthorized: %s", cause)
	if err != nil {
		log.Errorf("Writing http response: %v", err)
	}
}
