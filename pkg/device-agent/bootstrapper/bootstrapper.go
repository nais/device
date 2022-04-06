package bootstrapper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/bootstrap"
	"github.com/nais/device/pkg/ioconvenience"
)

func BootstrapDevice(ctx context.Context, deviceInfo *bootstrap.DeviceInfo, bootstrapAPI string, client *http.Client) (*bootstrap.Config, error) {
	deviceInfoURL := fmt.Sprintf("%s/api/v2/device/info", bootstrapAPI)
	err := postDeviceInfo(ctx, deviceInfoURL, deviceInfo, client)
	if err != nil {
		return nil, fmt.Errorf("posting device info: %w", err)
	}

	bootstrapConfigURL := fmt.Sprintf("%s/api/v2/device/config/%s", bootstrapAPI, deviceInfo.Serial)
	bootstrapConfig, err := getBootstrapConfig(ctx, bootstrapConfigURL, client)
	if err != nil {
		return nil, fmt.Errorf("getting bootstrap config: %w", err)
	}

	return bootstrapConfig, nil
}

func postDeviceInfo(ctx context.Context, url string, deviceInfo *bootstrap.DeviceInfo, client *http.Client) error {
	dib, err := json.Marshal(deviceInfo)
	if err != nil {
		return fmt.Errorf("marshaling device info: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(dib))
	if err != nil {
		return fmt.Errorf("make request: %w", err)
	}
	req.Header.Set("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("posting device info to bootstrap API (%v): %w", url, err)
	}

	defer ioconvenience.CloseWithLog(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		log.Warningf("bad response from bootstrap-api, request body: %v", string(body))
		return fmt.Errorf("bootstrap api (%v) returned status %v", url, resp.Status)
	}

	return nil
}

func getBootstrapConfig(ctx context.Context, url string, client *http.Client) (*bootstrap.Config, error) {
	get := func() (*bootstrap.Config, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		defer ioconvenience.CloseWithLog(resp.Body)

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("got statuscode %d from bootstrap api", resp.StatusCode)
		}

		bootstrapConfig := &bootstrap.Config{}
		err = json.NewDecoder(resp.Body).Decode(bootstrapConfig)
		if err != nil {
			return nil, err
		}

		return bootstrapConfig, nil
	}

	attempts := 3

	for i := 0; i < attempts; i++ {
		bootstrapConfig, err := get()
		if err != nil {
			log.Warnf("Attempt %d/%d at getting bootstrap config failed: %s", i+1, attempts, err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Debugf("Got bootstrap config from bootstrap api: %v", bootstrapConfig)
		return bootstrapConfig, nil
	}

	return nil, fmt.Errorf("unable to get boostrap config in %v attempts from %v", attempts, url)
}
