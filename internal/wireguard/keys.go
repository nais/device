package wireguard

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func ReadOrCreatePrivateKey(path string, log *logrus.Entry) (wgtypes.Key, error) {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return wgtypes.Key{}, fmt.Errorf("read private key: %w", err)
	}

	if errors.Is(err, fs.ErrNotExist) {
		log.Info("no private key found, generating new one...")
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			return wgtypes.Key{}, fmt.Errorf("generate private key: %w", err)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return wgtypes.Key{}, fmt.Errorf("create config dir: %w", err)
		}

		if err := os.WriteFile(path, []byte(key.String()), 0o600); err != nil {
			return wgtypes.Key{}, fmt.Errorf("write private key: %w", err)
		}

		return key, nil
	}

	log.Info("found private key, using it...")

	key, err := wgtypes.ParseKey(strings.TrimSpace(string(b)))
	if err != nil {
		return wgtypes.Key{}, fmt.Errorf("parse private key: %w", err)
	}

	return key, nil
}
