package wireguard_test

import (
	"testing"

	"github.com/nais/device/internal/device-agent/wireguard"
	"github.com/stretchr/testify/assert"
)

func TestWGGenKey(t *testing.T) {
	privateKey := wireguard.WgGenKey()
	assert.Len(t, privateKey, 32)
	privateKeyB64 := wireguard.KeyToBase64(privateKey)
	assert.Len(t, privateKeyB64, 44)
}
