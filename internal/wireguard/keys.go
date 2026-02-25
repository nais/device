package wireguard

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type PrivateKey []byte

func (p PrivateKey) Public() []byte {
	key, _ := wgtypes.NewKey(p)
	pub := key.PublicKey()
	return []byte(pub.String())
}

func (p PrivateKey) Private() []byte {
	key, _ := wgtypes.NewKey(p)
	return []byte(key.String())
}

func GenKey() (PrivateKey, error) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}
	return PrivateKey(key[:]), nil
}

func ReadOrCreatePrivateKey(path string, log *logrus.Entry) (PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	if errors.Is(err, fs.ErrNotExist) {
		log.Info("no private key found, generating new one...")
		b, err = GenKey()
		if err != nil {
			return nil, fmt.Errorf("generate private key: %w", err)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("create config dir: %w", err)
		}

		if err := os.WriteFile(path, b, 0o600); err != nil {
			return nil, fmt.Errorf("write private key: %w", err)
		}
	} else {
		log.Info("found private key, using it...")
	}

	return PrivateKey(b), nil
}
