package bootstrapper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/pkg/bootstrap"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func WatchEnrollments(ctx context.Context, db *database.APIServerDB, bootstrapApiURL, bootstrapApiCredentials string, publicKey []byte, apiEndpoint string) {
	for {
		select {
		case <-time.After(1 * time.Second):
			deviceInfos, err := fetchDeviceInfos(bootstrapApiURL, bootstrapApiCredentials)
			if err != nil {
				log.Errorf("Fetching device infos: %v", err)
			}

			for _, enrollment := range deviceInfos {
				err := db.AddDevice(ctx, enrollment.Owner, enrollment.PublicKey, enrollment.Serial, enrollment.Platform)
				if err != nil {
					log.Errorf("adding device: %v", err)
				}

				device, err := db.ReadDevice(enrollment.PublicKey)
				if err != nil {
					log.Errorf("getting device: %v", err)
				}

				err = pushBootstrapConfig(bootstrapApiURL, bootstrapApiCredentials, device.Serial, bootstrap.Config{
					DeviceIP:       device.IP,
					PublicKey:      string(publicKey),
					TunnelEndpoint: apiEndpoint,
					APIServerIP:    "10.255.240.1",
				})

				if err != nil {
					log.Errorf("pushing bootstrap config: %v", err)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func pushBootstrapConfig(bootstrapperURL, bootstrapperCredentials, serial string, bootstrapConfig bootstrap.Config) error {
	b, err := json.Marshal(bootstrapConfig)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	r, err := http.Post(bootstrapperURL+"/api/v1/bootstrapconfig/" + serial, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("posting bootstrap config: %w", err)
	}

	log.Infof("POST bootstrap config %v, resp: %v", bootstrapConfig, r)
	return nil
}

func fetchDeviceInfos(bootstrapperURL, bootstrapperCredentials string) ([]bootstrap.DeviceInfo, error) {
	r, err := http.Get(bootstrapperURL + "/api/v1/deviceinfo")
	if err != nil {
		return nil, fmt.Errorf("getting device infos: %w", err)
	}

	var deviceInfos []bootstrap.DeviceInfo
	json.NewDecoder(r.Body).Decode(&deviceInfos)
	return deviceInfos, nil
}
