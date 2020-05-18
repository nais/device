package device_agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/nais/device/device-agent/apiserver"
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

func New(cfg config.Config, client *http.Client) (*DeviceAgent, error) {
	deviceAgent := &DeviceAgent{
		Config: cfg,
		Client: client,
	}

	if err := deviceAgent.setup(); err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(deviceAgent.Config.WireGuardConfigPath, []byte(deviceAgent.BaseConfig), 0600); err != nil {
		return nil, fmt.Errorf("writing base WireGuard config to disk: %v", err)
	}

	return deviceAgent, nil
}

func (d *DeviceAgent) setup() error {
	var err error
	if d.PrivateKey, err = ensurePrivateKey(d.Config.PrivateKeyPath); err != nil {
		return fmt.Errorf("ensuring private key: %w", err)
	}

	if d.Serial, err = serial.GetDeviceSerial(); err != nil {
		return fmt.Errorf("getting device serial: %v", err)
	}

	//d.BootstrapConfig, err = ensureBootstrapConfig(); err != nil {
	//	return err..
	//}
	if err := filesExist(d.Config.BootstrapConfigPath); err != nil {

		enrollmentToken, err := apiserver.GenerateEnrollmentToken(d.Serial, d.Config.Platform, wireguard.KeyToBase64(wireguard.WGPubKey(d.PrivateKey)))
		if err != nil {
			return fmt.Errorf("generating enrollment token: %v", err)
		}

		return fmt.Errorf("\n---\nno bootstrap token present. Send 'naisdevice' your enrollment token on slack by copying and pasting this:\n/msg @naisdevice enroll %v\n\n", enrollmentToken)
	}

	bootstrapToken, err := ioutil.ReadFile(d.Config.BootstrapConfigPath)
	if err != nil {
		return fmt.Errorf("reading bootstrap token: %v", err)
	}

	if d.BootstrapConfig, err = apiserver.ParseBootstrapToken(string(bootstrapToken)); err != nil {
		return fmt.Errorf("parsing bootstrap config: %v", err)
	}

	d.BaseConfig = GenerateBaseConfig(d.PrivateKey, d.BootstrapConfig)

	return nil
}

func (d *DeviceAgent) publicKey() []byte {
	return wireguard.KeyToBase64(wireguard.WGPubKey(d.PrivateKey))
}

//TODO(jhrv): test
func ensurePrivateKey(keyPath string) ([]byte, error) {
	if err := FileMustExist(keyPath); os.IsNotExist(err) {
		if err := ioutil.WriteFile(keyPath, wireguard.KeyToBase64(wireguard.WgGenKey()), 0600); err != nil {
			return nil, fmt.Errorf("writing private key to disk: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("ensuring private key exists: %w", err)
	}

	privateKeyEncoded, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %v", err)
	}

	privateKey, err := wireguard.Base64toKey(privateKeyEncoded)
	if err != nil {
		return nil, fmt.Errorf("decoding private key: %v", err)
	}

	return privateKey, nil
}

func filesExist(files ...string) error {
	for _, file := range files {
		if err := FileMustExist(file); err != nil {
			return err
		}
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

type DeviceInfo struct {
	PublicKey []byte `json:"publicKey"`
	Serial    string `json:"serial"`
	Platform  string `json:"platform"`
}

func (d *DeviceAgent) EnsureBootstrapConfig() (*config.BootstrapConfig, error) {
	di := DeviceInfo{
		PublicKey: d.publicKey(),
		Serial:    d.Serial,
		Platform:  d.Config.Platform,
	}

	b, err := json.Marshal(&di)
	if err != nil {
		return nil, fmt.Errorf("marshaling device info: %w", err)
	}

	resp, err := http.Post(d.Config.BootstrapAPI+"/api/v1/deviceinfo", "application/json", bytes.NewReader(b))

	if err != nil {
		return nil, fmt.Errorf("posting device info to bootstrap API: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("bootstrap api returned status %v", resp.Status)
	}

	// successfully posted device info
	bootstrapConfig, err := getBootstrapConfig(d.Config.BootstrapAPI + "/api/v1/bootstrapconfig")
	if err != nil {
		return nil, fmt.Errorf("getting bootstrap config: %w", err)
	}

	return bootstrapConfig, nil
}

func getBootstrapConfig(url string) (*config.BootstrapConfig, error) {
	attempts := 3

	for i := 0; i < attempts; i++ {
		resp, err := http.Get(url)
		//if err != nil {
		//	return nil, fmt.Errorf("getting config from bootstrap API: %w", err)
		//}
		//
		//if resp.StatusCode != http.StatusOK {
		//
		//}
		if err == nil && resp.StatusCode == 200 {
			var bootstrapConfig config.BootstrapConfig
			if err := json.NewDecoder(resp.Body).Decode(&bootstrapConfig); err != nil {
				return &bootstrapConfig, nil
			}
		}
		time.Sleep(1 * time.Second)
		continue
	}
	return nil, fmt.Errorf("unable to get boostrap config in %v attempts", attempts)
}

func adminCommandContext(ctx context.Context, command string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "sudo", append([]string{command}, arg...)...)
}
