package ip

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("next ip is assigned based on largest ip in list", func(t *testing.T) {
		availableIP, err := ipAllocator.NextIP([]string{"fd75:568f:d24:1:0000:0000:0000:0002", "fd75:568f:d24:1:0000:0000:0000:0001"})
		assert.NoError(t, err)
		assert.Equal(t, "fd75:568f:0d24:0001:0000:0000:0000:0003", availableIP)
	})
}

func TestNextIPv6WithLegacyAddresses(t *testing.T) {
	prefix := netip.MustParsePrefix("2600:1900:4150:f548::/64")
	allocator := NewV6Allocator(&prefix)
	tests := []struct {
		name  string
		taken []string
		want  string
	}{
		{
			name:  "only legacy addresses",
			taken: []string{"fd75:568f:d24:6::11d"},
			want:  "2600:1900:4150:f548::1",
		},
		{
			name:  "mixed prefixes",
			taken: []string{"2600:1900:4150:f548::9", "fd75:568f:d24:6::11d"},
			want:  "2600:1900:4150:f548::a",
		},
		{
			name:  "numeric ordering with compressed addresses",
			taken: []string{"2600:1900:4150:f548::f", "2600:1900:4150:f548::10"},
			want:  "2600:1900:4150:f548::11",
		},
		{
			name:  "mixed compressed and expanded addresses",
			taken: []string{"2600:1900:4150:f548::2", "2600:1900:4150:f548:0000:0000:0000:0010"},
			want:  "2600:1900:4150:f548::11",
		},
		{
			name:  "unassigned addresses",
			taken: []string{"", "fd75:568f:d24:6::11d"},
			want:  "2600:1900:4150:f548::1",
		},
		{
			name:  "other global prefix",
			taken: []string{"2600:1900:4150:f549::10"},
			want:  "2600:1900:4150:f548::1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := allocator.NextIP(tt.taken)
			require.NoError(t, err)
			assert.Equal(t, netip.MustParseAddr(tt.want).StringExpanded(), got)
		})
	}
}

func TestNextIPv6ReservesAPIServerAddress(t *testing.T) {
	prefix := netip.MustParsePrefix("2600:1900:4150:f548::1/64")
	got, err := NewV6Allocator(&prefix).NextIP([]string{"fd75:568f:d24:6::11d", "2600:1900:4150:f548::"})
	require.NoError(t, err)
	assert.Equal(t, netip.MustParseAddr("2600:1900:4150:f548::2").StringExpanded(), got)
}

func TestNextIPv6Errors(t *testing.T) {
	t.Run("no prefix", func(t *testing.T) {
		_, err := NewV6Allocator(nil).NextIP(nil)
		require.ErrorContains(t, err, "no prefix specified")
	})
	t.Run("malformed allocated address", func(t *testing.T) {
		prefix := netip.MustParsePrefix("2600:1900:4150:f548::/64")
		_, err := NewV6Allocator(&prefix).NextIP([]string{"not-an-ip"})
		require.ErrorContains(t, err, "parse allocated IPv6 address")
	})
	t.Run("no room after apiserver address", func(t *testing.T) {
		prefix := netip.MustParsePrefix("2600:1900:4150:f548::1/128")
		_, err := NewV6Allocator(&prefix).NextIP(nil)
		require.ErrorContains(t, err, "no available IPv6 addresses")
	})
}

func TestNextIPv6Exhaustion(t *testing.T) {
	for _, cidr := range []string{"2600:1900:4150:f548::/126", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffc/126"} {
		t.Run(cidr, func(t *testing.T) {
			prefix := netip.MustParsePrefix(cidr)
			allocator := NewV6Allocator(&prefix)
			last := prefix.Addr().Next().Next().Next()
			_, err := allocator.NextIP([]string{last.String()})
			require.Error(t, err)
		})
	}
}
