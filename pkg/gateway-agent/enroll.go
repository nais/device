package gateway_agent

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/nais/device/pkg/device-agent/wireguard"
	"github.com/nais/device/pkg/passwordhash"
	"github.com/nais/device/pkg/pb"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/proto"
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
	password, err := passwordhash.RandomBytes(32)
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}

	salt, err := passwordhash.RandomBytes(16)
	if err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}

	key := passwordhash.HashPassword(password, salt)
	formatted := passwordhash.FormatHash(key, salt)

	req := &pb.EnrollGatewayRequest{
		Gateway: &pb.Gateway{
			Name:      e.cfg.Name,
			PublicKey: string(wireguard.PublicKey([]byte(e.cfg.PrivateKey))),
			Ip:        e.cfg.PublicIP,
		},
		Shadow: string(formatted),
	}

	payload, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("encode protobuf: %w", err)
	}

	err = e.persistPassword(password, DefaultConfigPath)
	if err != nil {
		return fmt.Errorf("persist password: %w", err)
	}

	fmt.Fprintf(os.Stderr, "API server password has been generated and written to %s.\n", DefaultConfigPath)
	fmt.Fprintf(os.Stderr, "Please run the following command to enroll this gateway with the API server:\n\n")
	fmt.Fprintf(os.Stderr, "controlplane-cli gateway enroll --request ")
	os.Stderr.Sync()

	_, err = base64.NewEncoder(base64.StdEncoding, os.Stdout).Write(payload)
	fmt.Fprintf(os.Stderr, "\n")

	return err
}

func (e *Enroller) persistPassword(password []byte, file string) error {
	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(password)
	_, err = fmt.Fprintf(fd, "GATEWAY_AGENT_APISERVERPASSWORD=\"%s\"\n", encoded)
	if err != nil {
		fd.Close()
		return fmt.Errorf("append to file: %w", err)
	}

	return fd.Close()
}
