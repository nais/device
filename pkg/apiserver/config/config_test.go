package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPrefixAddress(t *testing.T) {
	t.Run("test different for tenants", func(t *testing.T) {
		tests := []struct {
			expected string
			tenantId uint16
		}{
			{"fd75:568f:d24::", 0},
			{"fd75:568f:d24:1::", 1},
			{"fd75:568f:d24:5::", 5},
			{"fd75:568f:d24:ffff::", MaxTenantId},
		}

		for _, tt := range tests {
			actual := getWireGuardIPv6(tt.tenantId)
			assert.Equal(t, tt.expected, actual.Addr().String())
		}
	})
}
