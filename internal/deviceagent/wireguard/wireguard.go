package wireguard

import (
	"fmt"
	"os"
	"strings"

	"github.com/nais/device/internal/deviceagent/filesystem"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func EnsurePrivateKey(keyPath string) (wgtypes.Key, error) {
	if err := filesystem.FileMustExist(keyPath); os.IsNotExist(err) {
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return wgtypes.Key{}, fmt.Errorf("generating private key: %w", err)
		}
		if err := os.WriteFile(keyPath, []byte(key.String()), 0o600); err != nil {
			return wgtypes.Key{}, fmt.Errorf("writing private key to disk: %w", err)
		}
		return key, nil
	} else if err != nil {
		return wgtypes.Key{}, fmt.Errorf("ensuring private key exists: %w", err)
	}

	privateKeyEncoded, err := os.ReadFile(keyPath)
	if err != nil {
		return wgtypes.Key{}, fmt.Errorf("reading private key: %w", err)
	}

	if key, err := wgtypes.ParseKey(strings.TrimSpace(string(privateKeyEncoded))); err == nil {
		return key, nil
	}

	if len(privateKeyEncoded) != len(wgtypes.Key{}) {
		return wgtypes.Key{}, fmt.Errorf("parsing private key: invalid key length %d", len(privateKeyEncoded))
	}

	legacyKey, err := wgtypes.NewKey(privateKeyEncoded)
	if err != nil {
		return wgtypes.Key{}, fmt.Errorf("parsing legacy private key: %w", err)
	}

	if err := os.WriteFile(keyPath, []byte(legacyKey.String()), 0o600); err != nil {
		return wgtypes.Key{}, fmt.Errorf("rewriting legacy private key to disk: %w", err)
	}
	if err := os.Chmod(keyPath, 0o600); err != nil {
		return wgtypes.Key{}, fmt.Errorf("setting private key file mode: %w", err)
	}

	return legacyKey, nil
}
