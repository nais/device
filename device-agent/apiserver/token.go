package apiserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/nais/device/device-agent/config"
)

func GenerateEnrollmentToken(serial, platform string, publicKey []byte) (string, error) {
	type enrollmentConfig struct {
		Serial    string `json:"serial"`
		PublicKey string `json:"publicKey"`
		Platform  string `json:"platform"`
	}

	ec := enrollmentConfig{
		Serial:    serial,
		PublicKey: string(publicKey),
		Platform:  platform,
	}

	if b, err := json.Marshal(ec); err != nil {
		return "", fmt.Errorf("marshalling enrollment config: %w", err)
	} else {
		return base64.StdEncoding.EncodeToString(b), nil
	}
}

func ParseBootstrapToken(bootstrapToken string) (*config.BootstrapConfig, error) {
	b, err := base64.StdEncoding.DecodeString(bootstrapToken)
	if err != nil {
		return nil, fmt.Errorf("base64 decoding bootstrap token: %w", err)
	}

	var bootstrapConfig config.BootstrapConfig
	if err := json.Unmarshal(b, &bootstrapConfig); err != nil {
		return nil, fmt.Errorf("unmarshalling bootstrap token json: %w", err)
	}

	return &bootstrapConfig, nil
}
