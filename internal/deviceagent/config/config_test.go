package config_test

import (
	"testing"

	"github.com/nais/device/internal/deviceagent/config"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestJitaOAuth2ConfigFor(t *testing.T) {
	cfg := &config.Config{
		AzureOAuth2Config:  oauth2.Config{ClientID: "azure-login"},
		JitaOAuth2Config:   oauth2.Config{ClientID: "azure-jita"},
		GoogleOAuth2Config: oauth2.Config{ClientID: "google"},
	}

	assert.Equal(t, "azure-jita", cfg.JitaOAuth2ConfigFor(pb.AuthProvider_Azure).ClientID)
	assert.Equal(t, "google", cfg.JitaOAuth2ConfigFor(pb.AuthProvider_Google).ClientID)
}
