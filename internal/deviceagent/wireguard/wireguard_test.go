package wireguard_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestGenKey(t *testing.T) {
	key, err := wgtypes.GeneratePrivateKey()
	assert.NoError(t, err)
	assert.Len(t, key, 32)
	assert.Len(t, key.String(), 44)
}
