package google

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/internal/auth"
)

type handler struct {
	*auth.Handler
	allowedDomains []string
}

func New(ctx context.Context, config auth.Config) *handler {
	if len(config.AllowedDomains) == 0 {
		panic("google auth handler requires at least one allowed domain")
	}

	h, err := auth.New(ctx, config, jwt.WithRequiredClaim("hd"), jwt.WithRequiredClaim("email"))
	if err != nil {
		panic(fmt.Sprintf("google auth handler: %v", err))
	}

	return &handler{Handler: h, allowedDomains: config.AllowedDomains}
}
