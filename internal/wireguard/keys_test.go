package wireguard_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nais/device/internal/wireguard"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestReadOrCreatePrivateKey_CreatesKeyWhenMissing(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "private.key")

	key, err := wireguard.ReadOrCreatePrivateKey(keyPath, logrus.New().WithField("test", t.Name()))
	require.NoError(t, err)
	assert.NotEqual(t, wgtypes.Key{}, key)

	stored, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, key.String(), string(stored))

	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestReadOrCreatePrivateKey_MigratesLegacyRawKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "private.key")

	original, err := wgtypes.GeneratePrivateKey()
	require.NoError(t, err)

	legacyBytes := make([]byte, len(original))
	copy(legacyBytes, original[:])
	require.NoError(t, os.WriteFile(keyPath, legacyBytes, 0o644))

	key, err := wireguard.ReadOrCreatePrivateKey(keyPath, logrus.New().WithField("test", t.Name()))
	require.NoError(t, err)
	assert.Equal(t, original, key)

	migrated, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, original.String(), string(migrated))

	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
