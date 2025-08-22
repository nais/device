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
	var groups []string
	if err := token.Get(claimName, &groups); err != nil {
		return nil, fmt.Errorf("unable to get claim %q, %w", claimName, err)
	}

	return groups, nil
}
