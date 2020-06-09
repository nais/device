package runtimeconfig

import (
	"context"
	"fmt"
	"github.com/nais/device/pkg/bootstrap"
	log "github.com/sirupsen/logrus"
	"os"

	"github.com/nais/device/device-agent/auth"
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
	SessionID       auth.SessionID
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

	if FileExists(cfg.BootstrapConfigPath) {
		log.Infoln("Device already bootstrapped")
		rc.BootstrapConfig, err = bootstrapper.ReadFromFile(cfg.BootstrapConfigPath)
		if err != nil {
			return nil, fmt.Errorf("reading bootstrap config from file: %w", err)
		}
	} else {
		log.Infoln("Bootstrapping device")
		rc.BootstrapConfig, err = bootstrapper.ReadFromFile(cfg.BootstrapConfigPath)
		client, err := auth.AzureAuthenticatedClient(ctx, rc.Config.OAuth2Config)
		if err != nil {
			return nil, fmt.Errorf("authenticating with Azure: %w", err)
		}

		b := bootstrapper.New(
			wireguard.PublicKey(rc.PrivateKey),
			rc.Config.BootstrapConfigPath,
			rc.Serial,
			rc.Config.Platform,
			rc.Config.BootstrapAPI,
			client,
		)

		if rc.BootstrapConfig, err = b.EnsureBootstrapConfig(); err != nil {
			return nil, fmt.Errorf("unable to ensure bootstrap config: %w", err)
		}
	}

	if rc.SessionID, err = auth.RunFlow(ctx, cfg.ConfigDir, cfg.APIServer); err != nil {
		return nil, fmt.Errorf("getting session id: %w", err)
	}

	return rc, nil
}

func FileExists(filepath string) bool {
	info, err := os.Stat(filepath)
	if err != nil || info.IsDir() {
		return false
	}
	return true
}
