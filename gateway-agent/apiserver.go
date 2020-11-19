package gateway_agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type GatewayConfig struct {
	Devices []Device `json:"devices"`
	Routes  []string `json:"routes"`
}

type Device struct {
	PSK       string `json:"psk"`
	PublicKey string `json:"publicKey"`
	IP        string `json:"ip"`
}

func GetGatewayConfig(config Config, client http.Client) (*GatewayConfig, error) {
	gatewayConfigURL := fmt.Sprintf("http://%s/gatewayconfig", config.BootstrapConfig.APIServerIP)
	req, err := http.NewRequest(http.MethodGet, gatewayConfigURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting peer config from apiserver: %w", err)
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading bytes, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching gatewayConfig from apiserver: %v %v %v", resp.StatusCode, resp.Status, string(b))
	}

	var gatewayConfig GatewayConfig
	err = json.Unmarshal(b, &gatewayConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json from apiserver: bytes: %v, error: %w", string(b), err)
	}

	RegisteredDevices.Set(float64(len(gatewayConfig.Devices)))

	return &gatewayConfig, nil
}
