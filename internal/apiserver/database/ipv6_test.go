package database

import (
	"context"
	"fmt"
	"net/netip"
	"path/filepath"
	"testing"

	"github.com/nais/device/internal/apiserver/ip"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrollmentWithLegacyIPv6(t *testing.T) {
	tests := []struct {
		name     string
		devices  []string
		gateways []string
		want     string
	}{
		{
			name:    "only legacy device",
			devices: []string{"fd75:568f:d24:6::11d"},
			want:    "2600:1900:4150:f548::1",
		},
		{
			name:     "only legacy gateway",
			gateways: []string{"fd75:568f:d24:6::11d"},
			want:     "2600:1900:4150:f548::1",
		},
		{
			name:     "current device hidden by legacy gateway",
			devices:  []string{"2600:1900:4150:f548::9"},
			gateways: []string{"fd75:568f:d24:6::11d"},
			want:     "2600:1900:4150:f548::a",
		},
		{
			name:     "current gateway hidden by legacy device",
			devices:  []string{"fd75:568f:d24:6::11d"},
			gateways: []string{"2600:1900:4150:f548::9"},
			want:     "2600:1900:4150:f548::a",
		},
		{
			name:     "numeric maximum across both tables",
			devices:  []string{"2600:1900:4150:f548::f"},
			gateways: []string{"2600:1900:4150:f548::10"},
			want:     "2600:1900:4150:f548::11",
		},
		{
			name:    "current devices without gateways",
			devices: []string{"2600:1900:4150:f548::9"},
			want:    "2600:1900:4150:f548::a",
		},
		{
			name:     "current gateways without devices",
			gateways: []string{"2600:1900:4150:f548::9"},
			want:     "2600:1900:4150:f548::a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			prefix := netip.MustParsePrefix("2600:1900:4150:f548::/64")
			v4 := ip.NewV4Allocator(netip.MustParsePrefix("10.0.0.0/24"), []string{"10.0.0.1"})
			db, err := New(filepath.Join(t.TempDir(), "device.db"), v4, ip.NewV6Allocator(&prefix), false, logrus.New())
			require.NoError(t, err)
			conn := db.queries.(*Queries).db
			t.Cleanup(func() { require.NoError(t, conn.Close()) })

			for i, addr := range tt.devices {
				_, err := conn.Exec(`INSERT INTO devices (username, serial, platform, public_key, ipv4, ipv6)
					VALUES ('legacy-user', ?, 'linux', ?, ?, ?)`, fmt.Sprint(i), fmt.Sprintf("device-key-%d", i), fmt.Sprintf("10.0.0.%d", 10+i), addr)
				require.NoError(t, err)
			}
			for i, addr := range tt.gateways {
				_, err := conn.Exec(`INSERT INTO gateways (name, endpoint, public_key, ipv4, ipv6, password_hash)
					VALUES (?, 'gateway:51820', ?, ?, ?, 'hash')`, fmt.Sprintf("legacy-%d", i), fmt.Sprintf("gateway-key-%d", i), fmt.Sprintf("10.0.0.%d", 20+i), addr)
				require.NoError(t, err)
			}

			require.NoError(t, db.AddGateway(ctx, &pb.Gateway{Name: "new", PublicKey: "new-gateway-key"}))
			gateway, err := db.ReadGateway(ctx, "new")
			require.NoError(t, err)
			want := netip.MustParseAddr(tt.want)
			assert.Equal(t, want.StringExpanded(), gateway.Ipv6)

			require.NoError(t, db.AddDevice(ctx, &pb.Device{Serial: "new", Platform: "linux", PublicKey: "new-device-key"}))
			device, err := db.ReadDeviceBySerialPlatform(ctx, "new", "linux")
			require.NoError(t, err)
			assert.Equal(t, want.Next().StringExpanded(), device.Ipv6)

			for i, addr := range tt.devices {
				var stored string
				require.NoError(t, conn.QueryRow("SELECT ipv6 FROM devices WHERE serial = ?", fmt.Sprint(i)).Scan(&stored))
				assert.Equal(t, addr, stored)
			}
			for i, addr := range tt.gateways {
				gateway, err := db.ReadGateway(ctx, fmt.Sprintf("legacy-%d", i))
				require.NoError(t, err)
				assert.Equal(t, addr, gateway.Ipv6)
			}
		})
	}
}
