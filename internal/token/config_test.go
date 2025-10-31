package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		expectedErr string
	}{
		{
			name: "missing client ID",
			config: Config{
				Issuer:   "https://issuer.com",
				Endpoint: "https://endpoint.com",
			},
			wantErr:     true,
			expectedErr: "client ID is required",
		},
		{
			name: "Google issuer without allowed domains",
			config: Config{
				ClientID: "client-id",
				Issuer:   "https://accounts.google.com",
				Endpoint: "https://endpoint.com",
			},
			wantErr:     true,
			expectedErr: "at least one allowed domain is required for Google tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
