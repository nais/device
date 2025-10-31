package google

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/internal/token"
)

func (h *handler) ParseHeader(headers http.Header, header string) (*token.User, error) {
	if tok, err := jwt.ParseHeader(headers, header, h.opts...); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	} else {
		return h.tokenToUser(tok)
	}
}

func (h *handler) ParseString(str string) (*token.User, error) {
	if tok, err := jwt.ParseString(str, h.opts...); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	} else {
		return h.tokenToUser(tok)
	}
}

func (h *handler) tokenToUser(tok jwt.Token) (*token.User, error) {
	emailClaim, _ := tok.Get("email")
	email, _ := emailClaim.(string)
	if email == "" {
		return nil, fmt.Errorf("empty email claim in token")
	}

	subClaim, _ := tok.Get("sub")
	sub, _ := subClaim.(string)
	if sub == "" {
		return nil, fmt.Errorf("empty sub claim in token")
	}

	hdClaim, _ := tok.Get("hd")
	hd, _ := hdClaim.(string)

	if !slices.Contains(h.allowedDomains, hd) {
		return nil, fmt.Errorf("'%s' not in allowed domains: %v", hd, h.allowedDomains)
	}

	return &token.User{
		ID:     sub,
		Email:  email,
		Groups: []string{"allUsers"},
	}, nil
}
