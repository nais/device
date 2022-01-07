package gateway_agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/nais/device/pkg/apiserver/api"
	"github.com/nais/device/pkg/ioconvenience"
)

func GetGatewayConfig(config Config, client http.Client) (*api.GatewayConfig, error) {
	gatewayConfigURL := fmt.Sprintf("http://%s/gatewayconfig", config.BootstrapConfig.APIServerIP)
	req, err := http.NewRequest(http.MethodGet, gatewayConfigURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting peer config from apiserver: %w", err)
	}

	defer ioconvenience.CloseWithLog(resp.Body)

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading bytes, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching gatewayConfig from apiserver: %v %v %v", resp.StatusCode, resp.Status, string(b))
	}

	gatewayConfig := &api.GatewayConfig{}
	err = json.Unmarshal(b, &gatewayConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json from apiserver: bytes: %v, error: %w", string(b), err)
	}

	RegisteredDevices.Set(float64(len(gatewayConfig.Devices)))

	return gatewayConfig, nil
}
