package auth

import (
	"fmt"
	"strings"

	"github.com/lestrrat-go/jwx/v3/jwt"
)

// StringClaim returns the value of a string claim from a JWT token. If the claim is not present or empty, it returns an
// error.
func StringClaim(claimName string, token jwt.Token) (string, error) {
	var value string
	if err := token.Get(claimName, &value); err != nil {
		return "", fmt.Errorf("unable to get claim %q, %w", claimName, err)
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("empty claim %s", claimName)
	}

	return value, nil
}

// GroupsClaim returns the value of the "groups" claim from a JWT token.
func GroupsClaim(token jwt.Token) ([]string, error) {
	const claimName = "groups"
	var raw []any
	if err := token.Get(claimName, &raw); err != nil {
		return nil, fmt.Errorf("unable to get claim %q, %w", claimName, err)
	}

	var groups []string
	for _, g := range raw {
		if gs, ok := g.(string); ok {
			groups = append(groups, gs)
		} else {
			return nil, fmt.Errorf("invalid group claim type: %T", g)
		}
	}

	return groups, nil
}
