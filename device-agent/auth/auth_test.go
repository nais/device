package auth_test

import (
	"testing"
	"time"

	"github.com/nais/device/device-agent/auth"
	"github.com/nais/device/device-agent/runtimeconfig"
	"github.com/stretchr/testify/assert"
)

func TestSessionInfo_Expired(t *testing.T) {
	expired := auth.SessionInfo{Expiry: 1}
	assert.True(t, expired.Expired())

	rc := runtimeconfig.RuntimeConfig{SessionInfo: nil}
	assert.True(t, rc.SessionInfo.Expired())

	valid := auth.SessionInfo{Expiry: time.Now().Unix() + 10}
	assert.False(t, valid.Expired())
	assert.False(t, valid.Expired())
}
