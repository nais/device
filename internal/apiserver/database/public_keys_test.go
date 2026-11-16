package database

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/nais/device/internal/wireguard"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestNew_RepairsRawPublicKeys(t *testing.T) {
	key := wgtypes.Key{0: '\n', 1: 0xff, 2: 0x80, 31: ' '}
	canonical := key.String()
	payload, err := json.Marshal(map[string]string{"wireguard_public_key": canonical})
	require.NoError(t, err)
	var legacyRequest struct {
		PublicKey []byte `json:"wireguard_public_key"`
	}
	require.NoError(t, json.Unmarshal(payload, &legacyRequest))

	tests := []struct {
		name   string
		stored string
		want   string
	}{
		{name: "new enrollment decoded by legacy receiver", stored: string(legacyRequest.PublicKey), want: canonical},
		{name: "canonical", stored: canonical, want: canonical},
		{name: "other lengths unchanged", stored: "invalid", want: "invalid"},
		{name: "double encoded unchanged", stored: base64.StdEncoding.EncodeToString([]byte(canonical)), want: base64.StdEncoding.EncodeToString([]byte(canonical))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "device.db")
			log := logrus.New()
			require.NoError(t, runMigrations(path, log))
			conn, err := sql.Open("sqlite3", path)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, conn.Close()) })

			_, err = conn.Exec(`INSERT INTO devices (username, serial, platform, public_key, ipv4)
				VALUES ('user', 'serial', 'linux', ?, '10.0.0.2')`, tt.stored)
			require.NoError(t, err)
			_, err = conn.Exec(`INSERT INTO gateways (name, endpoint, public_key, ipv4, password_hash)
				VALUES ('gateway', 'gateway:51820', ?, '10.0.0.3', 'hash')`, tt.stored)
			require.NoError(t, err)

			for range 2 {
				db, err := New(path, nil, nil, false, log)
				require.NoError(t, err)
				peers, err := db.ReadPeers(context.Background())
				require.NoError(t, err)
				require.Len(t, peers, 1)
				assert.Equal(t, tt.want, peers[0].GetPublicKey())

				gateway, err := db.ReadGateway(context.Background(), "gateway")
				require.NoError(t, err)
				assert.Equal(t, tt.want, gateway.PublicKey)

				if tt.want == canonical {
					cfg := wireguard.Config{Peers: wireguard.CastPeerList(peers)}
					var ini bytes.Buffer
					require.NoError(t, cfg.MarshalINI(&ini))
					assert.Contains(t, ini.String(), "PublicKey = "+canonical+"\n")
					parsed, err := wgtypes.ParseKey(peers[0].GetPublicKey())
					require.NoError(t, err)
					assert.Equal(t, key, parsed)
				}

				var stored string
				require.NoError(t, conn.QueryRow("SELECT public_key FROM devices").Scan(&stored))
				assert.Equal(t, tt.want, stored)
				require.NoError(t, conn.QueryRow("SELECT public_key FROM gateways").Scan(&stored))
				assert.Equal(t, tt.want, stored)
			}
		})
	}
}

func TestRepairRawPublicKeys_RollsBackOnConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device.db")
	log := logrus.New()
	require.NoError(t, runMigrations(path, log))
	conn, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })

	key := wgtypes.Key{0: 0xff}
	raw := string(key[:])
	_, err = conn.Exec(`INSERT INTO devices (username, serial, platform, public_key, ipv4)
		VALUES ('user', 'serial', 'linux', ?, '10.0.0.2')`, raw)
	require.NoError(t, err)
	_, err = conn.Exec(`INSERT INTO gateways (name, endpoint, public_key, ipv4, password_hash)
		VALUES ('raw', 'raw:51820', ?, '10.0.0.3', 'hash'),
		       ('canonical', 'canonical:51820', ?, '10.0.0.4', 'hash')`, raw, key.String())
	require.NoError(t, err)

	err = repairRawPublicKeys(conn, log)
	require.ErrorContains(t, err, "repair gateways public key")
	var stored string
	require.NoError(t, conn.QueryRow("SELECT public_key FROM devices").Scan(&stored))
	assert.Equal(t, raw, stored)
	require.NoError(t, conn.QueryRow("SELECT public_key FROM gateways WHERE name = 'raw'").Scan(&stored))
	assert.Equal(t, raw, stored)
}
