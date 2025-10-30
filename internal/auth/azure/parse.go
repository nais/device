package azure

import (
	"fmt"
	"net/http"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/internal/auth"
)

func (h *handler) ParseHeader(headers http.Header, header string) (*auth.User, error) {
	return h.Handler.ParseHeader(headers, header, h.validateToken)
}

func (h *handler) ParseString(token string) (*auth.User, error) {
	fmt.Printf("Parsing token: %s\n", token)
	return h.Handler.ParseString(token, h.validateToken)
}

func (h *handler) validateToken(token jwt.Token) (*auth.User, error) {
	tokenOID, ok := token.Get("oid")
	if !ok {
		return nil, fmt.Errorf("missing oid claim in token")
	}
	id := tokenOID.(string)

	var tokenEmail any
	tokenEmail, ok = token.Get("preferred_username")
	if !ok {
		tokenEmail, ok = token.Get("unique_name")
		if !ok {
			tokenEmail, ok = token.Get("upn")
			if !ok {
				return nil, fmt.Errorf("missing preferred_username, unique_name and upn claims in token")
			}
		}
	}
	email := tokenEmail.(string)

	groups := []string{"allUsers"}
	tokenGroups, ok := token.Get("groups")
	if !ok {
		return nil, fmt.Errorf("missing groups claim in token")
	}
	for _, group := range tokenGroups.([]any) {
		groups = append(groups, group.(string))
	}

	return &auth.User{
		ID:     id,
		Email:  email,
		Groups: groups,
	}, nil
}
