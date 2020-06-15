package runtimeconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/bootstrapper"
	"github.com/nais/device/pkg/bootstrap"
	log "github.com/sirupsen/logrus"

	"github.com/nais/device/device-agent/auth"
	"github.com/nais/device/device-agent/config"
	"github.com/nais/device/device-agent/serial"
	"github.com/nais/device/device-agent/wireguard"
)

type RuntimeConfig struct {
	Serial          string
	BootstrapConfig *bootstrap.Config
	Config          config.Config
	PrivateKey      []byte
	SessionInfo     *auth.SessionInfo
	Gateways        apiserver.Gateways
}

func (rc *RuntimeConfig) GetGateways() apiserver.Gateways {
	if rc == nil {
		return make(apiserver.Gateways, 0)
	}
	return rc.Gateways
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
		return nil, fmt.Errorf("getting device serial: %w", err)
	}

	if rc.BootstrapConfig, err = ensureBootstrapping(rc, ctx); err != nil {
		return nil, fmt.Errorf("ensuring bootstrap: %w", err)
	}

	return rc, nil
}

func ensureBootstrapping(rc *RuntimeConfig, ctx context.Context) (*bootstrap.Config, error) {
	if fileExists(rc.Config.BootstrapConfigPath) {
		log.Infoln("Device already bootstrapped")
		return readBootstrapConfigFromFile(rc.Config.BootstrapConfigPath)
	}

	log.Infoln("Bootstrapping device")
	client, err := auth.AzureAuthenticatedClient(ctx, rc.Config.OAuth2Config)
	if err != nil {
		return nil, fmt.Errorf("authenticating with Azure: %w", err)
	}

	cfg, err := bootstrapper.BootstrapDevice(
		&bootstrap.DeviceInfo{
			PublicKey: string(wireguard.PublicKey(rc.PrivateKey)),
			Serial:    rc.Serial,
			Platform:  rc.Config.Platform,
		},
		rc.Config.BootstrapAPI,
		client,
	)

	if err != nil {
		return nil, err
	}

	err = writeToJSONFile(rc.BootstrapConfig, rc.Config.BootstrapConfigPath)
	if err != nil {
		return nil, fmt.Errorf("writing bootstrap config to disk: %w", err)
	}

	return cfg, nil
}

func fileExists(filepath string) bool {
	info, err := os.Stat(filepath)
	if err != nil || info.IsDir() {
		return false
	}
	return true
}

func writeToJSONFile(strct interface{}, path string) error {
	b, err := json.Marshal(&strct)
	if err != nil {
		return fmt.Errorf("marshaling struct into json: %w", err)
	}
	if err := ioutil.WriteFile(path, b, 0600); err != nil {
		return err
	}
	return nil
}

func readBootstrapConfigFromFile(bootstrapConfigPath string) (*bootstrap.Config, error) {
	var bc bootstrap.Config
	b, err := ioutil.ReadFile(bootstrapConfigPath)
	if err != nil {
		return nil, fmt.Errorf("reading bootstrap config from disk: %w", err)
	}
	if err := json.Unmarshal(b, &bc); err != nil {
		return nil, fmt.Errorf("unmarshaling bootstrap config: %w", err)
	}
	return &bc, nil
}
