package gateway_agent

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/pbkdf2"
)

const (
	DefaultConfigPath = "/etc/default/gateway-agent"
)

type Enroller struct {
	cfg Config
}

func NewEnroller(cfg Config) Enroller {
	return Enroller{
		cfg: cfg,
	}
}

func (e *Enroller) Enroll(c *cli.Context) error {
	password, err := generatePassword()
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}
	salt, err := generatePassword()
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}
	//err = e.persistPassword(password)
	if err != nil {
		return fmt.Errorf("persist password: %w", err)
	}

	hash := HashPassword(password, salt)

	fmt.Print(string(hash))

	// req := pb.EnrollGatewayRequest{
	// Gateway: &pb.Gateway{
	// Name:      e.cfg.Name,
	// PublicKey: string(wireguard.PublicKey([]byte(e.cfg.PrivateKey))),
	// Ip:        e.cfg.PublicIP,
	// },
	// PasswordHash: string(hash),
	// }

	return nil
}

func HashPassword(password, salt string) []byte {
	buf := &bytes.Buffer{}
	key := base64.StdEncoding.EncodeToString(
		pbkdf2.Key([]byte(password), []byte(salt), 128069, 64, sha1.New),
	)
	fmt.Fprintf(buf, "$1$%s$%s", salt, key)
	return buf.Bytes()
}

func (e *Enroller) persistPassword(password string) error {
	fd, err := os.OpenFile(DefaultConfigPath, os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	_, err = fmt.Fprintf(fd, "GATEWAY_AGENT_APISERVERPASSWORD=%v", password)
	if err != nil {
		return fmt.Errorf("append to file: %w", err)
	}

	return nil
}

func generatePassword() (string, error) {
	bytes := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
