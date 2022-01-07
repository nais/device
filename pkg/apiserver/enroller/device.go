package enroller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nais/device/pkg/ioconvenience"
	"github.com/nais/device/pkg/pb"

	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/bootstrap"
)

func (e *Enroller) WatchDeviceEnrollments(ctx context.Context) {
	for {
		select {
		case <-time.After(1 * time.Second):
			if err := e.EnrollDevice(ctx); err != nil {
				log.Errorf("Enrolling devices: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (e *Enroller) EnrollDevice(ctx context.Context) error {
	deviceInfos, err := e.fetchDeviceInfos(e.BootstrapAPIURL)
	if err != nil {
		return fmt.Errorf("bootstrap: Fetching device infos: %v", err)
	}

	for _, enrollment := range deviceInfos {
		err := e.DB.AddDevice(ctx, &pb.Device{
			Username:  enrollment.Owner,
			PublicKey: enrollment.PublicKey,
			Serial:    enrollment.Serial,
			Platform:  enrollment.Platform,
		})

		if err != nil {
			return fmt.Errorf("bootstrap: Adding device: %v", err)
		}

		device, err := e.DB.ReadDevice(enrollment.PublicKey)
		if err != nil {
			return fmt.Errorf("bootstrap: Getting device: %v", err)
		}

		bootstrapConfig := bootstrap.Config{
			DeviceIP:       device.Ip,
			PublicKey:      e.APIServerPublicKey,
			TunnelEndpoint: e.APIServerEndpoint,
			APIServerIP:    "10.255.240.1",
		}

		err = e.postDeviceConfig(device.Serial, bootstrapConfig)

		if err != nil {
			return fmt.Errorf("bootstrap: Pushing bootstrap config: %v", err)
		}

		log.Infof("bootstrap: Bootstrapped device: %+v", bootstrapConfig)
	}

	return nil
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

	defer ioconvenience.CloseWithLog(r.Body)

	var deviceInfos []bootstrap.DeviceInfo
	err = json.NewDecoder(r.Body).Decode(&deviceInfos)
	if err != nil {
		return nil, fmt.Errorf("decoding deviceInfos: %w", err)
	}

	return deviceInfos, nil
}
