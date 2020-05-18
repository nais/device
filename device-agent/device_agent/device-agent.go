package device_agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"github.com/nais/device/device-agent/apiserver"
	"github.com/nais/device/device-agent/azure"
	"github.com/nais/device/device-agent/config"
	"github.com/nais/device/device-agent/serial"
	"github.com/nais/device/device-agent/wireguard"
	log "github.com/sirupsen/logrus"
)

type DeviceAgent struct {
	Serial          string
	BootstrapConfig *config.BootstrapConfig
	Config          config.Config
	PrivateKey      []byte
	Client          *http.Client
	BaseConfig      string
}

func New(cfg config.Config) (*DeviceAgent, error) {
	deviceAgent := &DeviceAgent{Config: cfg}
	if err := deviceAgent.setup(); err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(deviceAgent.Config.WireGuardConfigPath, []byte(deviceAgent.BaseConfig), 0600); err != nil {
		return nil, fmt.Errorf("writing base WireGuard config to disk: %v", err)
	}

	return deviceAgent, nil
}

//TODO(jhrv): needs refactor
func (d *DeviceAgent) setup() error {
	d.setPlatformDefaults()
	if err := d.platformPrerequisites(); err != nil {
		return fmt.Errorf("verifying platform prerequisites: %v", err)
	}

	if err := filesExist(d.Config.WireGuardBinary); err != nil {
		return fmt.Errorf("verifying if file exists: %v", err)
	}

	if err := ensureDirectories(d.Config.ConfigDir); err != nil {
		return fmt.Errorf("ensuring directory exists: %v", err)
	}

	if err := ensureKey(d.Config.PrivateKeyPath); err != nil {
		return fmt.Errorf("ensuring private key exists: %v", err)
	}

	privateKeyEncoded, err := ioutil.ReadFile(d.Config.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("reading private key: %v", err)
	}

	if d.PrivateKey, err = wireguard.Base64toKey(privateKeyEncoded); err != nil {
		return fmt.Errorf("decoding private key: %v", err)
	}

	if d.Serial, err = serial.GetDeviceSerial(); err != nil {
		return fmt.Errorf("getting device serial: %v", err)
	}

	if err := filesExist(d.Config.BootstrapTokenPath); err != nil {
		enrollmentToken, err := apiserver.GenerateEnrollmentToken(d.Serial, d.Config.Platform, wireguard.KeyToBase64(wireguard.WGPubKey(d.PrivateKey)))
		if err != nil {
			return fmt.Errorf("generating enrollment token: %v", err)
		}

		return fmt.Errorf("\n---\nno bootstrap token present. Send 'naisdevice' your enrollment token on slack by copying and pasting this:\n/msg @naisdevice enroll %v\n\n", enrollmentToken)
	}

	bootstrapToken, err := ioutil.ReadFile(d.Config.BootstrapTokenPath)
	if err != nil {
		return fmt.Errorf("reading bootstrap token: %v", err)
	}

	if d.BootstrapConfig, err = apiserver.ParseBootstrapToken(string(bootstrapToken)); err != nil {
		return fmt.Errorf("parsing bootstrap config: %v", err)
	}

	d.BaseConfig = GenerateBaseConfig(d.PrivateKey, d.BootstrapConfig)

	return nil
}

func ensureKey(keyPath string) error {
	if err := FileMustExist(keyPath); os.IsNotExist(err) {
		return ioutil.WriteFile(keyPath, wireguard.KeyToBase64(wireguard.WgGenKey()), 0600)
	} else if err != nil {
		return err
	}

	return nil
}

func filesExist(files ...string) error {
	for _, file := range files {
		if err := FileMustExist(file); err != nil {
			return err
		}
	}

	return nil
}

func ensureDirectories(dirs ...string) error {
	for _, dir := range dirs {
		if err := ensureDirectory(dir); err != nil {
			return err
		}
	}

	return nil
}

func ensureDirectory(dir string) error {
	info, err := os.Stat(dir)

	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0700)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%v is a file", dir)
	}

	return nil
}

func FileMustExist(filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%v is a directory", filepath)
	}

	return nil
}

func (d *DeviceAgent) AuthUser(ctx context.Context) error {
	token, err := azure.RunAuthFlow(ctx, d.Config.OAuth2Config)
	if err != nil {
		return fmt.Errorf("unable to get token for user: %v", err)
	}

	d.Client = d.Config.OAuth2Config.Client(ctx, token)
	return nil
}

func (d *DeviceAgent) SyncConfig() error {
	gateways, err := apiserver.GetGateways(d.Client, d.Config.APIServer, d.Serial)

	if err != nil {
		log.Errorf("Unable to get gateway config: %v", err)
	}

	wireGuardPeers := wireguard.GenerateWireGuardPeers(gateways)

	if err := ioutil.WriteFile(d.Config.WireGuardConfigPath, []byte(d.BaseConfig+wireGuardPeers), 0600); err != nil {
		log.Errorf("Writing WireGuard config to disk: %v", err)
	}

	log.Debugf("Wrote WireGuard config to disk")
	return nil
}

func adminCommandContext(ctx context.Context, command string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "sudo", append([]string{command}, arg...)...)
}
