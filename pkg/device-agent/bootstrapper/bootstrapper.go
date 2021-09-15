package bootstrapper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/nais/device/pkg/bootstrap"
	log "github.com/sirupsen/logrus"
)

func BootstrapDevice(deviceInfo *bootstrap.DeviceInfo, bootstrapAPI string, client *http.Client) (*bootstrap.Config, error) {
	deviceInfoURL := fmt.Sprintf("%s/api/v2/device/info", bootstrapAPI)
	err := postDeviceInfo(deviceInfoURL, deviceInfo, client)
	if err != nil {
		return nil, fmt.Errorf("posting device info: %w", err)
	}

	bootstrapConfigURL := fmt.Sprintf("%s/api/v2/device/config/%s", bootstrapAPI, deviceInfo.Serial)
	bootstrapConfig, err := getBootstrapConfig(bootstrapConfigURL, client)
	if err != nil {
		return nil, fmt.Errorf("getting bootstrap config: %w", err)
	}

	return bootstrapConfig, nil
}

func postDeviceInfo(url string, deviceInfo *bootstrap.DeviceInfo, client *http.Client) error {
	dib, err := json.Marshal(deviceInfo)
	if err != nil {
		return fmt.Errorf("marshaling device info: %w", err)
	}

	resp, err := client.Post(url, "application/json", bytes.NewReader(dib))

	if err != nil {
		return fmt.Errorf("posting device info to bootstrap API (%v): %w", url, err)
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
