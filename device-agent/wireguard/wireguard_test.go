package wireguard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWGGenKey(t *testing.T) {
	privateKey := WgGenKey()
	assert.Len(t, privateKey, 32)
	privateKeyB64 := KeyToBase64(privateKey)
	assert.Len(t, privateKeyB64, 44)
}
