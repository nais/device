package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/internal/token"
	"github.com/sirupsen/logrus"
)

type handler struct {
	log  *logrus.Entry
	opts []jwt.ParseOption
}

var _ token.Parser = &handler{}

func New(ctx context.Context, log *logrus.Entry, config token.Config) token.Parser {
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("azure token parser: %v", err))
	}

	ksp, err := token.NewKSP(ctx, config.Endpoint)
	if err != nil {
		panic(fmt.Sprintf("azure token parser: %v", err))
	}

	return &handler{
		log: log,
		opts: []jwt.ParseOption{
			jwt.WithValidate(true),
			jwt.InferAlgorithmFromKey(true),
			jwt.WithKeySetProvider(ksp),
			jwt.WithAcceptableSkew(5 * time.Second),
			jwt.WithIssuer(config.Issuer),
			jwt.WithAudience(config.ClientID),
		},
	}
}
