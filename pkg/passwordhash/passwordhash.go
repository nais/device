package passwordhash

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

func HashPassword(password, salt []byte) []byte {
	const iterations = 200029
	const keysize = 64
	return pbkdf2.Key(password, salt, iterations, keysize, sha256.New)
}

func FormatHash(key, salt []byte) []byte {
	buf := &bytes.Buffer{}

	buf.WriteString("$1$")
	buf.WriteString(base64.StdEncoding.EncodeToString(salt))
	buf.WriteString("$")
	buf.WriteString(base64.StdEncoding.EncodeToString(key))

	return buf.Bytes()
}

// Validate a password hash string.
// Format: "$<VERSION>$<SALT>$<KEY>"
// where VERSION = 1, and SALT and KEY are base64 standard-encoded strings.
func Validate(password, hash []byte) error {
	parts := strings.SplitN(string(hash), "$", 4)

	if len(parts) != 4 || len(parts[0]) != 0 || parts[1] != "1" {
		return fmt.Errorf("hash format error")
	}

	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("hash format error")
	}

	key, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return fmt.Errorf("hash format error")
	}

	x := HashPassword(password, salt)
	if !bytes.Equal(x, key) {
		return fmt.Errorf("invalid password")
	}

	return nil
}

func RandomBytes(length int) ([]byte, error) {
	buf := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func GeneratePasswordAndHash() (password, hash string, err error) {
	passwordBytes, err := RandomBytes(32)
	if err != nil {
		return "", "", fmt.Errorf("generate password: %w", err)
	}

	password = base64.StdEncoding.EncodeToString(passwordBytes)

	salt, err := RandomBytes(16)
	if err != nil {
		return "", "", fmt.Errorf("generate salt: %w", err)
	}

	key := HashPassword([]byte(password), salt)
	passhash := FormatHash(key, salt)
	return password, string(passhash), nil
}
