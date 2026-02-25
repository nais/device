package wireguard_test

import (
	"testing"

	"github.com/nais/device/internal/wireguard"
	"github.com/stretchr/testify/assert"
)

func TestGenKey(t *testing.T) {
	privateKey, err := wireguard.GenKey()
	assert.NoError(t, err)
	assert.Len(t, privateKey, 32)
	privateKeyB64 := wireguard.KeyToBase64(privateKey)
	assert.Len(t, privateKeyB64, 44)
}
