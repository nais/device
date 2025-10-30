package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
)

type Config struct {
	ClientID       string
	Issuer         string
	Endpoint       string
	AllowedDomains []string
}

type Handler struct {
	jwks    *jwk.AutoRefresh
	options []jwt.ParseOption
	ctx     context.Context
	Config
}

// New creates a new auth handler based on the provided configuration and JWT parse options.
// For the default use cases use of google.New and azure.New is recommended.
func New(ctx context.Context, config Config, parseOptions ...jwt.ParseOption) (*Handler, error) {
	if len(config.ClientID) == 0 {
		return nil, fmt.Errorf("client ID is required")
	}
	if len(config.Issuer) == 0 {
		return nil, fmt.Errorf("issuer is required")
	}
	if len(config.Endpoint) == 0 {
		return nil, fmt.Errorf("endpoint is required")
	}

	h := &Handler{
		jwks:    jwk.NewAutoRefresh(context.Background()),
		ctx:     ctx,
		Config:  config,
		options: parseOptions,
	}

	if err := h.setup(ctx); err != nil {
		return nil, err
	}

	return h, nil
}

func (h *Handler) setup(ctx context.Context) error {
	ar := jwk.NewAutoRefresh(ctx)
	ar.Configure(h.Endpoint, jwk.WithMinRefreshInterval(time.Hour))

	_, err := ar.Refresh(ctx, h.Endpoint)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}

	h.jwks = ar
	return nil
}

// KeySetFrom ignores the provided token and fetches the JWKS from the configured endpoint. This makes sense as we have one handler per issuer.
func (h *Handler) KeySetFrom(_ jwt.Token) (jwk.Set, error) {
	ctx, cancel := context.WithTimeout(h.ctx, 10*time.Second)
	defer cancel()

	return h.jwks.Fetch(ctx, h.Endpoint)
}

func (h *Handler) opts() []jwt.ParseOption {
	return append([]jwt.ParseOption{
		jwt.WithValidate(true),
		jwt.InferAlgorithmFromKey(true),
		jwt.WithKeySetProvider(h),
		jwt.WithAcceptableSkew(5 * time.Second),
		jwt.WithIssuer(h.Issuer),
		jwt.WithAudience(h.ClientID),
	}, h.options...)
}

func (h *Handler) ParseHeader(headers http.Header, header string, validate func(jwt.Token) (*User, error)) (*User, error) {
	if tok, err := jwt.ParseHeader(headers, header, h.opts()...); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	} else {
		return validate(tok)
	}
}

func (h *Handler) ParseString(token string, validate func(jwt.Token) (*User, error)) (*User, error) {
	if tok, err := jwt.ParseString(token, h.opts()...); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	} else {
		return validate(tok)
	}
}
