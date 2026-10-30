package enroll

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestNormalizeWireGuardPublicKey(t *testing.T) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)

	canonical := privateKey.PublicKey().String()
	legacyEncoded := base64.StdEncoding.EncodeToString([]byte(canonical))

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "canonical key",
			input: canonical,
			want:  canonical,
		},
		{
			name:  "legacy base64 encoded canonical key",
			input: legacyEncoded,
			want:  canonical,
		},
		{
			name:    "malformed key",
			input:   "not-a-wireguard-key",
			wantErr: "invalid wireguard public key: expected canonical key or base64-encoded canonical key",
		},
		{
			name:    "empty key",
			input:   " ",
			wantErr: "invalid wireguard public key: empty value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeWireGuardPublicKey(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
