package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Gateway struct {
	PublicKey string   `json:"publicKey"`
	Endpoint  string   `json:"endpoint"`
	IP        string   `json:"ip"`
	Routes    []string `json:"routes"`
}

func GetGateways(client *http.Client, apiServerURL, serial string) ([]Gateway, error) {
	deviceConfigAPI := fmt.Sprintf("%s/devices/%s/gateways", apiServerURL, serial)
	r, err := client.Get(deviceConfigAPI)
	if err != nil {
		return nil, fmt.Errorf("getting device config: %w", err)
	}
	defer r.Body.Close()

	var gateways []Gateway
	if err := json.NewDecoder(r.Body).Decode(&gateways); err != nil {
		return nil, fmt.Errorf("unmarshalling response body into gateways: %w", err)
	}

	return gateways, nil
}
