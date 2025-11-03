package token

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
)

type KeySetProvider struct {
	ctx      context.Context
	endpoint string
	jwks     *jwk.AutoRefresh
}

func NewKSP(ctx context.Context, jwksURL string) (*KeySetProvider, error) {
	jwks := jwk.NewAutoRefresh(ctx)
	jwks.Configure(jwksURL, jwk.WithMinRefreshInterval(time.Hour))
	_, err := jwks.Refresh(ctx, jwksURL)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}

	return &KeySetProvider{ctx: ctx, jwks: jwks, endpoint: jwksURL}, nil
}

// KeySetFrom ignores the provided token and fetches the JWKS from the configured endpoint. This makes sense as we have one handler per issuer.
func (ksp *KeySetProvider) KeySetFrom(tok jwt.Token) (jwk.Set, error) {
	ctx, cancel := context.WithTimeout(ksp.ctx, 10*time.Second)
	defer cancel()

	return ksp.jwks.Fetch(ctx, ksp.endpoint)
}
