package ip

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindAvailableIPv6(t *testing.T) {
	prefix := netip.MustParsePrefix("fd75:568f:d24:1::/64")
	ipAllocator := NewV6Allocator(&prefix)

	t.Run("finds the lowest ip address in range", func(t *testing.T) {
		availableIP, err := ipAllocator.NextIP(nil)
		assert.NoError(t, err)
		assert.Equal(t, "fd75:568f:0d24:0001:0000:0000:0000:0001", availableIP)
	})

	t.Run("uses next ip when last used is passed in", func(t *testing.T) {
		availableIP, err := ipAllocator.NextIP([]string{"fd75:568f:d24:1:0000:0000:0000:0001"})
		assert.NoError(t, err)
		assert.Equal(t, "fd75:568f:0d24:0001:0000:0000:0000:0002", availableIP)

		availableIP, err = ipAllocator.NextIP([]string{"fd75:568f:d24:1:0000:0000:0000:0003"})
		assert.NoError(t, err)
		assert.Equal(t, "fd75:568f:0d24:0001:0000:0000:0000:0004", availableIP)
	})

	t.Run("next ip is assigned based on largest ip in list (sorted as string)", func(t *testing.T) {
		availableIP, err := ipAllocator.NextIP([]string{"fd75:568f:d24:1:0000:0000:0000:0002", "fd75:568f:d24:1:0000:0000:0000:0001"})
		assert.NoError(t, err)
		assert.Equal(t, "fd75:568f:0d24:0001:0000:0000:0000:0003", availableIP)
	})
}
