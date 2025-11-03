package google

import (
	"testing"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/internal/token"
	"github.com/stretchr/testify/assert"
)

func TestHandler_TokenToUser(t *testing.T) {
	tests := []struct {
		name           string
		allowedDomains []string
		setupToken     func() jwt.Token
		expectError    bool
		expectedErr    string
		expectedUser   *token.User
	}{
		{
			name:           "valid token with allowed domain",
			allowedDomains: []string{"example.com", "test.com"},
			setupToken: func() jwt.Token {
				tok := jwt.New()
				if err := tok.Set("email", "user@example.com"); err != nil {
					t.Fatalf("failed to set email claim: %v", err)
				}
				if err := tok.Set("sub", "google-user-123"); err != nil {
					t.Fatalf("failed to set sub claim: %v", err)
				}
				if err := tok.Set("hd", "example.com"); err != nil {
					t.Fatalf("failed to set hd claim: %v", err)
				}
				return tok
			},
			expectError: false,
			expectedUser: &token.User{
				ID:     "google-user-123",
				Email:  "user@example.com",
				Groups: []string{"allUsers"},
			},
		},
		{
			name:           "domain not in allowed domains",
			allowedDomains: []string{"example.com", "test.com"},
			setupToken: func() jwt.Token {
				tok := jwt.New()
				if err := tok.Set("email", "user@forbidden.com"); err != nil {
					t.Fatalf("failed to set email claim: %v", err)
				}
				if err := tok.Set("sub", "google-user-123"); err != nil {
					t.Fatalf("failed to set sub claim: %v", err)
				}
				if err := tok.Set("hd", "forbidden.com"); err != nil {
					t.Fatalf("failed to set hd claim: %v", err)
				}
				return tok
			},
			expectError: true,
			expectedErr: "'forbidden.com' not in allowed domains: [example.com test.com]",
		},
		{
			name:           "missing hd claim",
			allowedDomains: []string{"example.com"},
			setupToken: func() jwt.Token {
				tok := jwt.New()
				if err := tok.Set("email", "user@example.com"); err != nil {
					t.Fatalf("failed to set email claim: %v", err)
				}
				if err := tok.Set("sub", "google-user-123"); err != nil {
					t.Fatalf("failed to set sub claim: %v", err)
				}
				return tok
			},
			expectError: true,
			expectedErr: "'' not in allowed domains: [example.com]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{
				allowedDomains: tt.allowedDomains,
			}

			tok := tt.setupToken()
			user, err := h.tokenToUser(tok)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, user)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedUser, user)
			}
		})
	}
}
