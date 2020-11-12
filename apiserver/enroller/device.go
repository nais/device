package enroller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/pkg/bootstrap"
	log "github.com/sirupsen/logrus"
	"time"
)

func (e *Enroller) WatchDeviceEnrollments(ctx context.Context) {
	for {
		select {
		case <-time.After(1 * time.Second):
			deviceInfos, err := e.fetchDeviceInfos(e.BootstrapAPIURL)
			if err != nil {
				log.Errorf("bootstrap: Fetching device infos: %v", err)
				continue
			}

			for _, enrollment := range deviceInfos {
				err := e.DB.AddDevice(ctx, database.Device{
					Username:  enrollment.Owner,
					PublicKey: enrollment.PublicKey,
					Serial:    enrollment.Serial,
					Platform:  enrollment.Platform,
				})

				if err != nil {
					log.Errorf("bootstrap: Adding device: %v", err)
					continue
				}

				device, err := e.DB.ReadDevice(enrollment.PublicKey)
				if err != nil {
					log.Errorf("bootstrap: Getting device: %v", err)
					continue
				}

				bootstrapConfig := bootstrap.Config{
					DeviceIP:       device.IP,
					PublicKey:      e.APIServerPublicKey,
					TunnelEndpoint: e.APIServerEndpoint,
					APIServerIP:    "10.255.240.1",
				}

				err = e.postDeviceConfig(device.Serial, bootstrapConfig)

				if err != nil {
					log.Errorf("bootstrap: Pushing bootstrap config: %v", err)
					continue
				}

				log.Infof("bootstrap: Bootstrapped device: %+v", bootstrapConfig)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (e *Enroller) postDeviceConfig(serial string, bootstrapConfig bootstrap.Config) error {
	b, err := json.Marshal(bootstrapConfig)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	r, err := e.Client.Post(fmt.Sprintf("%s/api/v2/device/config/%s", e.BootstrapAPIURL, serial), "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("posting bootstrap config: %w", err)
	}

	log.Infof("bootstrap: Bootstrapped %+v, response: %v", bootstrapConfig, r)
	return nil
}

func (e *Enroller) fetchDeviceInfos(bootstrapURL string) ([]bootstrap.DeviceInfo, error) {
	r, err := e.Client.Get(bootstrapURL + "/api/v2/device/info")
	if err != nil {
		return nil, fmt.Errorf("getting device infos: %w", err)
	}

	var deviceInfos []bootstrap.DeviceInfo
	err = json.NewDecoder(r.Body).Decode(&deviceInfos)
	if err != nil {
		return nil, fmt.Errorf("decoding deviceInfos: %w", err)
	}

	return deviceInfos, nil
}
