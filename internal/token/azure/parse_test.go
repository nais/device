package azure

import (
	"testing"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/nais/device/internal/token"
	"github.com/stretchr/testify/assert"
)

func TestHandler_TokenToUser(t *testing.T) {
	h := &handler{}

	tests := []struct {
		name         string
		setupToken   func() jwt.Token
		expectError  bool
		expectedErr  string
		expectedUser *token.User
	}{
		{
			name: "valid token with preferred_username",
			setupToken: func() jwt.Token {
				tok := jwt.New()
				tok.Set("oid", "user-oid-123")
				tok.Set("preferred_username", "user@example.com")
				tok.Set("groups", []interface{}{"group1", "group2"})
				return tok
			},
			expectError: false,
			expectedUser: &token.User{
				ID:     "user-oid-123",
				Email:  "user@example.com",
				Groups: []string{"allUsers", "group1", "group2"},
			},
		},
		{
			name: "fallback to unique_name",
			setupToken: func() jwt.Token {
				tok := jwt.New()
				tok.Set("oid", "user-oid-456")
				tok.Set("unique_name", "user2@example.com")
				tok.Set("groups", []interface{}{"group3"})
				return tok
			},
			expectError: false,
			expectedUser: &token.User{
				ID:     "user-oid-456",
				Email:  "user2@example.com",
				Groups: []string{"allUsers", "group3"},
			},
		},
		{
			name: "fallback to upn",
			setupToken: func() jwt.Token {
				tok := jwt.New()
				tok.Set("oid", "user-oid-789")
				tok.Set("upn", "user3@example.com")
				tok.Set("groups", []interface{}{})
				return tok
			},
			expectError: false,
			expectedUser: &token.User{
				ID:     "user-oid-789",
				Email:  "user3@example.com",
				Groups: []string{"allUsers"},
			},
		},
		{
			name: "missing all email claims",
			setupToken: func() jwt.Token {
				tok := jwt.New()
				tok.Set("oid", "user-oid-123")
				tok.Set("groups", []interface{}{"group1"})
				return tok
			},
			expectError: true,
			expectedErr: "missing preferred_username, unique_name and upn claims in token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
