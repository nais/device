package gateway_agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nais/device/device-agent/wireguard"
	"github.com/nais/device/pkg/bootstrap"
	"github.com/nais/device/pkg/secretmanager"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

const enrollmentTokenPrefix = "enrollment-token"

type SecretManager interface {
	GetSecret(string) (*secretmanager.Secret, error)
}

type Bootstrapper struct {
	SecretManager SecretManager
	Config        *Config
	HTTPClient    *http.Client
}

func (b *Bootstrapper) EnsureBootstrapConfig() (*bootstrap.Config, error) {
	bootstrapConfig, err := readBootstrapConfigFromFile(b.Config.BootstrapConfigPath)

	if bootstrapConfig != nil && err == nil {
		return bootstrapConfig, nil
	}

	if err != nil {
		log.Infof("Attempted to read bootstrap config: %v", err)
	}

	gatewayInfo := &bootstrap.GatewayInfo{
		Name:      b.Config.Name,
		PublicIP:  b.Config.PublicIP,
		PublicKey: string(wireguard.PublicKey([]byte(b.Config.PrivateKey))),
	}
	bc, err := BootstrapGateway(gatewayInfo, b.Config.BootstrapApiURL, b.HTTPClient)
	if err != nil {
		return nil, fmt.Errorf("bootstrapping gateway: %w", err)
	}

	if err := writeToJSONFile(bc, b.Config.BootstrapConfigPath); err != nil {
		return nil, fmt.Errorf("writing bootstrap config to file: %w", err)
	}

	return bc, nil
}

func BootstrapGateway(gatewayInfo *bootstrap.GatewayInfo, bootstrapAPI string, client *http.Client) (*bootstrap.Config, error) {
	gatewayInfoUrl := fmt.Sprintf("%s/api/v2/gateway/info", bootstrapAPI)
	err := postGatewayInfo(gatewayInfoUrl, gatewayInfo, client)
	if err != nil {
		return nil, fmt.Errorf("posting device info: %w", err)
	}

	bootstrapConfigURL := fmt.Sprintf("%s/api/v2/gateway/config/%s", bootstrapAPI, gatewayInfo.Name)
	bootstrapConfig, err := getBootstrapConfig(bootstrapConfigURL, client)
	if err != nil {
		return nil, fmt.Errorf("getting bootstrap config: %w", err)
	}

	return bootstrapConfig, nil
}

func postGatewayInfo(url string, gatewayInfo *bootstrap.GatewayInfo, client *http.Client) error {
	dib, err := json.Marshal(gatewayInfo)
	if err != nil {
		return fmt.Errorf("marshaling device info: %w", err)
	}

	resp, err := client.Post(url, "application/json", bytes.NewReader(dib))
	if err != nil {
		return fmt.Errorf("posting info to bootstrap API (%v): %w", url, err)
	}

	if resp.StatusCode != http.StatusCreated {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		log.Warningf("bad response from bootstrap-api, request body: %v", string(body))
		return fmt.Errorf("bootstrap api (%v) returned status %v", url, resp.Status)
	}

	return nil
}

func getBootstrapConfig(url string, client *http.Client) (*bootstrap.Config, error) {
	attempts := 3

	for i := 0; i < attempts; i++ {
		resp, err := client.Get(url)

		if err == nil && resp.StatusCode == 200 {
			var bootstrapConfig bootstrap.Config
			if err := json.NewDecoder(resp.Body).Decode(&bootstrapConfig); err == nil {
				log.Debugf("Got bootstrap config from bootstrap api: %v", bootstrapConfig)
				return &bootstrapConfig, nil
			}
		}
		time.Sleep(1 * time.Second)
		continue
	}
	return nil, fmt.Errorf("unable to get boostrap config in %v attempts from %v", attempts, url)
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
