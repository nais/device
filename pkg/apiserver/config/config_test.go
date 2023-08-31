package config

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPrefixAddress(t *testing.T) {
	t.Run("test different for tenants", func(t *testing.T) {
		assert.Equal(t, "fd75:568f:d24::", getWireGuardIPv6(0).Addr().String())
		assert.Equal(t, "fd75:568f:d24:1::", getWireGuardIPv6(1).Addr().String())
		assert.Equal(t, "fd75:568f:d24:5::", getWireGuardIPv6(5).Addr().String())
		assert.Equal(t, "fd75:568f:d24:ffff::", getWireGuardIPv6(uint16(math.Pow(2, 16)-1)).Addr().String())
	})
}
