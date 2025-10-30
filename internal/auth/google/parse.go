package google

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/internal/auth"
)

func (h *handler) ParseHeader(headers http.Header, header string) (*auth.User, error) {
	return h.Handler.ParseHeader(headers, header, h.validateToken)
}

func (h *handler) ParseString(token string) (*auth.User, error) {
	return h.Handler.ParseString(token, h.validateToken)
}

func (h *handler) validateToken(token jwt.Token) (*auth.User, error) {
	emailClaim, _ := token.Get("email")
	email, _ := emailClaim.(string)
	if email == "" {
		return nil, fmt.Errorf("empty email claim in token")
	}

	subClaim, _ := token.Get("sub")
	sub, _ := subClaim.(string)
	if sub == "" {
		return nil, fmt.Errorf("empty sub claim in token")
	}

	hdClaim, _ := token.Get("hd")
	hd, _ := hdClaim.(string)

	if !slices.Contains(h.allowedDomains, hd) {
		return nil, fmt.Errorf("'%s' not in allowed domains: %v", hd, h.allowedDomains)
	}

	return &auth.User{
		ID:     sub,
		Email:  email,
		Groups: []string{"allUsers"},
	}, nil
}
