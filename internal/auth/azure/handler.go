package azure

import (
	"context"
	"fmt"

	"github.com/nais/device/internal/auth"
)

type handler struct {
	*auth.Handler
}

var _ auth.TokenParser = &handler{}

func New(ctx context.Context, config auth.Config) *handler {
	h, err := auth.New(ctx, config)
	if err != nil {
		panic(fmt.Sprintf("azure auth handler: %v", err))
	}

	return &handler{h}
}
