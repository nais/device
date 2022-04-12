package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/curve25519"
)

type PrivateKey []byte

// Public returns the public key base64 encoded
func (p PrivateKey) Public() []byte {
	return keyToBase64(pubKey(p))
}

// Private returns the private key base64 encoded
func (p PrivateKey) Private() []byte {
	return keyToBase64(p)
}

func GenKey() (PrivateKey, error) {
	var privateKey [32]byte

	n, err := rand.Read(privateKey[:])

	if err != nil || n != len(privateKey) {
		return nil, fmt.Errorf("unable to generate random bytes")
	}

	privateKey[0] &= 248
	privateKey[31] = (privateKey[31] & 127) | 64
	return PrivateKey(privateKey[:]), nil
}

func pubKey(privateKeySlice []byte) []byte {
	var privateKey [32]byte
	var publicKey [32]byte
	copy(privateKey[:], privateKeySlice[:])

	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return publicKey[:]
}

func keyToBase64(key []byte) []byte {
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(key)))
	base64.StdEncoding.Encode(dst, key)
	return dst
}

func ReadOrCreatePrivateKey(path string, log *logrus.Entry) (PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	if errors.Is(err, fs.ErrNotExist) {
		log.Info("No private key found, generating new one...")
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
		log.Info("Found private key, using it...")
	}

	return PrivateKey(b), nil
}
