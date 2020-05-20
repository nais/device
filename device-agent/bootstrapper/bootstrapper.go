package bootstrapper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nais/device/pkg/bootstrap"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type bootstrapper struct {
	Client              *http.Client
	BootstrapAPI        string
	BootstrapConfigPath string
	DeviceInfo          bootstrap.DeviceInfo
}

func New(publicKey []byte, bootstrapConfigPath, serial, platform, bootstrapAPI string, client *http.Client) *bootstrapper {
	return &bootstrapper{
		Client:              client,
		BootstrapAPI:        bootstrapAPI,
		BootstrapConfigPath: bootstrapConfigPath,
		DeviceInfo: bootstrap.DeviceInfo{
			PublicKey: string(publicKey),
			Serial:    serial,
			Platform:  platform,
		},
	}
}

func (b *bootstrapper) EnsureBootstrapConfig() (*bootstrap.Config, error) {
	if FileExists(b.BootstrapConfigPath) {
		return readFromFile(b.BootstrapConfigPath)
	}

	bootstrapConfig, err := b.BootstrapDevice()
	if err != nil {
		return nil, fmt.Errorf("bootstrapping device: %w", err)
	}

	if err := writeToFile(bootstrapConfig, b.BootstrapConfigPath); err != nil {
		return nil, fmt.Errorf("writing bootstrap config to disk: %w", err)
	}

	return bootstrapConfig, nil
}

func (b *bootstrapper) BootstrapDevice() (*bootstrap.Config, error) {
	dib, err := json.Marshal(b.DeviceInfo)
	if err != nil {
		return nil, fmt.Errorf("marshaling device info: %w", err)
	}

	resp, err := http.Post(fmt.Sprintf("%s/api/v1/deviceinfo", b.BootstrapAPI), "application/json", bytes.NewReader(dib))

	if err != nil {
		return nil, fmt.Errorf("posting device info to bootstrap API: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("bootstrap api returned status %v", resp.Status)
	}

	bootstrapConfig, err := getBootstrapConfig(fmt.Sprintf("%s/api/v1/bootstrapconfig/%s", b.BootstrapAPI, b.DeviceInfo.Serial))
	if err != nil {
		return nil, fmt.Errorf("getting bootstrap config: %w", err)
	}

	return bootstrapConfig, nil
}

func getBootstrapConfig(url string) (*bootstrap.Config, error) {
	attempts := 3

	for i := 0; i < attempts; i++ {
		resp, err := http.Get(url)

		if err == nil && resp.StatusCode == 200 {
			var bootstrapConfig bootstrap.Config
			if err := json.NewDecoder(resp.Body).Decode(&bootstrapConfig); err == nil {
				return &bootstrapConfig, nil
			}
		}
		time.Sleep(1 * time.Second)
		continue
	}
	return nil, fmt.Errorf("unable to get boostrap config in %v attempts", attempts)
}

func FileExists(filepath string) bool {
	info, err := os.Stat(filepath)
	if err != nil || info.IsDir() {
		return false
	}
	return true
}

func writeToFile(bootstrapConfig *bootstrap.Config, bootstrapConfigPath string) error {
	b, err := json.Marshal(bootstrapConfig)
	if err != nil {
		return fmt.Errorf("marshaling bootstrap config: %w", err)
	}
	if err := ioutil.WriteFile(bootstrapConfigPath, b, 0600); err != nil {
		return err
	}
	return nil
}

func readFromFile(bootstrapConfigPath string) (*bootstrap.Config, error) {
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
