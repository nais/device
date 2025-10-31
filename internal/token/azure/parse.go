package azure

import (
	"fmt"
	"net/http"

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
	tokenOID, ok := tok.Get("oid")
	if !ok {
		return nil, fmt.Errorf("missing oid claim in token")
	}
	id := tokenOID.(string)

	var tokenEmail any
	tokenEmail, ok = tok.Get("preferred_username")
	if !ok {
		tokenEmail, ok = tok.Get("unique_name")
		if !ok {
			tokenEmail, ok = tok.Get("upn")
			if !ok {
				return nil, fmt.Errorf("missing preferred_username, unique_name and upn claims in token")
			}
		}
	}
	email := tokenEmail.(string)

	groups := []string{"allUsers"}
	tokenGroups, ok := tok.Get("groups")
	if !ok {
		return nil, fmt.Errorf("missing groups claim in token")
	}
	for _, group := range tokenGroups.([]any) {
		groups = append(groups, group.(string))
	}

	return &token.User{
		ID:     id,
		Email:  email,
		Groups: groups,
	}, nil
}
