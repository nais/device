package google

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/internal/token"
)

type handler struct {
	allowedDomains []string
	opts           []jwt.ParseOption
}

func New(ctx context.Context, config token.Config) token.Parser {
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("google token parser: %v", err))
	}

	ksp, err := token.NewKSP(ctx, config.Endpoint)
	if err != nil {
		panic(fmt.Sprintf("google token parser: %v", err))
	}

	return &handler{
		allowedDomains: config.AllowedDomains,
		opts: []jwt.ParseOption{
			jwt.WithValidate(true),
			jwt.InferAlgorithmFromKey(true),
			jwt.WithKeySetProvider(ksp),
			jwt.WithAcceptableSkew(5 * time.Second),
			jwt.WithIssuer(config.Issuer),
			jwt.WithAudience(config.ClientID),
			jwt.WithRequiredClaim("hd"),
			jwt.WithRequiredClaim("email"),
		},
	}
}
