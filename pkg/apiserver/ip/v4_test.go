package ip

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindAvailableIP(t *testing.T) {
	t.Run("finds the lowest ip address in range", func(t *testing.T) {
		ipAllocator := NewV4Allocator(netip.MustParsePrefix("10.0.0.0/30"), nil)
		availableIP, err := ipAllocator.NextIP([]string{"10.0.0.2"})
		assert.NoError(t, err)
		assert.Equal(t, "10.0.0.1", availableIP)
	})

	t.Run("doesn't give out reserved ips", func(t *testing.T) {
		ipAllocator := NewV4Allocator(netip.MustParsePrefix("10.0.0.0/30"), []string{"10.0.0.1"})
		availableIP, err := ipAllocator.NextIP(nil)
		assert.NoError(t, err)
		assert.Equal(t, "10.0.0.2", availableIP)
	})

	t.Run("fills gaps", func(t *testing.T) {
		ipAllocator := NewV4Allocator(netip.MustParsePrefix("10.0.0.0/30"), nil)
		availableIP, err := ipAllocator.NextIP([]string{"10.0.0.1", "10.0.0.3"})
		assert.NoError(t, err)
		assert.Equal(t, "10.0.0.2", availableIP)
	})

	t.Run("returns an error if noone is available", func(t *testing.T) {
		ipAllocator := NewV4Allocator(netip.MustParsePrefix("10.0.0.1/30"), nil)
		_, err := ipAllocator.NextIP([]string{"10.0.0.1", "10.0.0.2"})
		assert.Error(t, err)
	})
}
