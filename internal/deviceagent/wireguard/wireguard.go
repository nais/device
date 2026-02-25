package wireguard

import (
	"encoding/base64"
	"fmt"
	"os"

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
		return wgtypes.Key{}, fmt.Errorf("reading private key: %v", err)
	}

	var key wgtypes.Key
	b, err := base64.StdEncoding.DecodeString(string(privateKeyEncoded))
	if err != nil {
		return wgtypes.Key{}, fmt.Errorf("decoding private key: %v", err)
	}
	copy(key[:], b)

	return key, nil
}
