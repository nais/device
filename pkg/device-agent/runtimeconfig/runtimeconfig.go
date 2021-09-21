package runtimeconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/bootstrap"
	"github.com/nais/device/pkg/device-agent/auth"
	"github.com/nais/device/pkg/device-agent/bootstrapper"
	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/serial"
	"github.com/nais/device/pkg/device-agent/wireguard"
	"github.com/nais/device/pkg/pb"
)

type RuntimeConfig struct {
	Serial          string
	BootstrapConfig *bootstrap.Config
	Config          *config.Config
	PrivateKey      []byte
	SessionInfo     *pb.Session
}

func New(cfg *config.Config) (*RuntimeConfig, error) {
	rc := &RuntimeConfig{
		Config: cfg,
	}

	var err error

	if rc.PrivateKey, err = wireguard.EnsurePrivateKey(rc.Config.PrivateKeyPath); err != nil {
		return nil, fmt.Errorf("ensuring private key: %w", err)
	}

	if rc.Serial, err = serial.GetDeviceSerial(cfg.SerialPath); err != nil {
		return nil, fmt.Errorf("getting device serial: %w", err)
	}

	rc.BootstrapConfig, err = readBootstrapConfigFromFile(rc.Config.BootstrapConfigPath)
	if err != nil {
		log.Infof("Unable to read bootstrap config from file: %v", err)
	} else {
		log.Infof("Read bootstrap config from file: %v", rc.Config.BootstrapConfigPath)
	}

	log.Infof("Runtime config initialized with public key: %s", wireguard.PublicKey(rc.PrivateKey))

	return rc, nil
}

func EnsureBootstrapping(rc *RuntimeConfig, ctx context.Context) (*bootstrap.Config, error) {
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

	err = writeToJSONFile(cfg, rc.Config.BootstrapConfigPath)
	if err != nil {
		return nil, fmt.Errorf("writing bootstrap config to disk: %w", err)
	}

	return cfg, nil
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
