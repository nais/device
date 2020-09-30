package runtimeconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/bootstrapper"
	"github.com/nais/device/pkg/bootstrap"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"

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

// UpdateGateways sets a slice of gateways on the RuntimeConfig but preserves the previous healthstatus
func (rc *RuntimeConfig) UpdateGateways(new apiserver.Gateways) {
	old := rc.Gateways
	previousHealthState := func(name string, gws apiserver.Gateways) bool {
		for _, gw := range gws {
			if gw.Name == name {
				return gw.Healthy
			}
		}
		return false
	}

	for _, gw := range new {
		gw.Healthy = previousHealthState(gw.Name, old)
	}

	rc.Gateways = new
}

// Write configuration file into a Writer
func (rc *RuntimeConfig) Write(w io.Writer) (int, error) {
	var written int
	baseConfig := wireguard.GenerateBaseConfig(rc.BootstrapConfig, rc.PrivateKey)
	wt, err := w.Write([]byte(baseConfig))
	written += wt
	if err != nil {
		return written, err
	}

	wireGuardPeers := rc.Gateways.MarshalIni()
	wt, err = w.Write(wireGuardPeers)
	written += wt

	return written, err
}

func New(cfg config.Config, ctx context.Context) (*RuntimeConfig, error) {
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
