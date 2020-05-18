package bootstrapper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type BootstrapConfig struct {
	TunnelIP    string `json:"deviceIP"`
	PublicKey   string `json:"publicKey"`
	Endpoint    string `json:"tunnelEndpoint"`
	APIServerIP string `json:"apiServerIP"`
}

type bootstrapper struct {
	Client              *http.Client
	BootstrapAPI        string
	BootstrapConfigPath string
	DeviceInfo          deviceInfo
}

type deviceInfo struct {
	PublicKey []byte `json:"publicKey"`
	Serial    string `json:"serial"`
	Platform  string `json:"platform"`
}

func New(publicKey []byte, bootstrapConfigPath, serial, platform, bootstrapAPI string, client *http.Client) *bootstrapper {
	return &bootstrapper{
		Client:              client,
		BootstrapAPI:        bootstrapAPI,
		BootstrapConfigPath: bootstrapConfigPath,
		DeviceInfo: deviceInfo{
			PublicKey: publicKey,
			Serial:    serial,
			Platform:  platform,
		},
	}
}

func (b *bootstrapper) EnsureBootstrapConfig() (*BootstrapConfig, error) {
	// if file exists, unmarshal into struct and return. Else ....

	dib, err := json.Marshal(b.DeviceInfo)
	if err != nil {
		return nil, fmt.Errorf("marshaling device info: %w", err)
	}

	resp, err := http.Post(b.BootstrapAPI+"/api/v1/deviceinfo", "application/json", bytes.NewReader(dib))

	if err != nil {
		return nil, fmt.Errorf("posting device info to bootstrap API: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("bootstrap api returned status %v", resp.Status)
	}

	// successfully posted device info
	bootstrapConfig, err := getBootstrapConfig(b.BootstrapAPI + "/api/v1/bootstrapconfig")
	if err != nil {
		return nil, fmt.Errorf("getting bootstrap config: %w", err)
	}

	return bootstrapConfig, nil
}

func getBootstrapConfig(url string) (*BootstrapConfig, error) {
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
			var bootstrapConfig BootstrapConfig
			if err := json.NewDecoder(resp.Body).Decode(&bootstrapConfig); err != nil {
				return &bootstrapConfig, nil
			}
		}
		time.Sleep(1 * time.Second)
		continue
	}
	return nil, fmt.Errorf("unable to get boostrap config in %v attempts", attempts)
}
