package runtimeconfig_test

import (
	"github.com/nais/device/device-agent/auth"
	"github.com/nais/device/device-agent/config"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestRuntimeConfigDevMode(t *testing.T) {
	t.Run("sessionInfo is nil when devmode is false", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.PrivateKeyPath = os.TempDir() + "private.key"
		cfg.DevMode = false
		config, err := runtimeconfig.New(cfg)
		assert.NoError(t, err)
		assert.Nil(t, config.SessionInfo)
	})

	t.Run("sessionInfo is populated when devmode is true", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.PrivateKeyPath = os.TempDir() + "private.key"
		cfg.DevMode = true
		config, err := runtimeconfig.New(cfg)
		assert.NoError(t, err)
		assert.Equal(t, &auth.SessionInfo{Key: "sessionkey", Expiry: 9999999999}, config.SessionInfo)
	})
}
