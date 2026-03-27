package iputil

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePrefix(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    netip.Prefix
		wantErr bool
	}{
		{
			name:  "IPv4 CIDR",
			input: "10.0.0.0/24",
			want:  netip.MustParsePrefix("10.0.0.0/24"),
		},
		{
			name:  "IPv4 host CIDR",
			input: "10.43.0.60/32",
			want:  netip.MustParsePrefix("10.43.0.60/32"),
		},
		{
			name:  "bare IPv4 address",
			input: "10.43.0.60",
			want:  netip.MustParsePrefix("10.43.0.60/32"),
		},
		{
			name:  "IPv6 CIDR",
			input: "fd01::/64",
			want:  netip.MustParsePrefix("fd01::/64"),
		},
		{
			name:  "bare IPv6 address",
			input: "fd00::1",
			want:  netip.MustParsePrefix("fd00::1/128"),
		},
		{
			name:  "CIDR with surrounding whitespace",
			input: " 10.0.0.0/24 ",
			want:  netip.MustParsePrefix("10.0.0.0/24"),
		},
		{
			name:  "bare IP with surrounding whitespace",
			input: " 10.43.0.60 ",
			want:  netip.MustParsePrefix("10.43.0.60/32"),
		},
		{
			name:    "garbage input",
			input:   "not-an-ip",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePrefix(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
