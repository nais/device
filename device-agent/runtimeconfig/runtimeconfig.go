package runtimeconfig

import (
	"context"
	"fmt"
	"github.com/nais/device/pkg/bootstrap"
	"net/http"

	"github.com/nais/device/device-agent/azure"
	"github.com/nais/device/device-agent/bootstrapper"
	"github.com/nais/device/device-agent/config"
	"github.com/nais/device/device-agent/serial"
	"github.com/nais/device/device-agent/wireguard"
)

type RuntimeConfig struct {
	Serial          string
	BootstrapConfig *bootstrap.Config
	Config          config.Config
	PrivateKey      []byte
	Client          *http.Client
}

func New(cfg config.Config, ctx context.Context) (*RuntimeConfig, error) {
	rc := &RuntimeConfig{
		Config: cfg,
	}

	var err error

	if rc.PrivateKey, err = wireguard.EnsurePrivateKey(rc.Config.PrivateKeyPath); err != nil {
		return nil, fmt.Errorf("ensuring private key: %w", err)
	}

	if rc.Serial, err = serial.GetDeviceSerial(); err != nil {
		return nil, fmt.Errorf("getting device serial: %v", err)
	}

	if rc.Client, err = azure.EnsureClient(ctx, cfg.OAuth2Config); err != nil {
		return nil, fmt.Errorf("ensuring authenticated http client: %w", err)
	}

	b := bootstrapper.New(
		wireguard.PublicKey(rc.PrivateKey),
		rc.Config.BootstrapConfigPath,
		rc.Serial,
		rc.Config.Platform,
		rc.Config.BootstrapAPI,
		rc.Client,
	)

	if rc.BootstrapConfig, err = b.EnsureBootstrapConfig(); err != nil {
		return nil, fmt.Errorf("unable to ensure bootstrap config: %w", err)
	}

	return rc, nil
}
