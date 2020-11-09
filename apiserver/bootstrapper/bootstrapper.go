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
	"strings"
	"time"
)

type bootstrapper struct {
	Client *http.Client
}

type BasicAuthTransport struct {
	Username string
	Password string
}

func (bat *BasicAuthTransport) Client() *http.Client {
	return &http.Client{Transport: bat}
}

func (bat BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(bat.Username, bat.Password)
	return http.DefaultTransport.RoundTrip(req)
}

func WatchEnrollments(ctx context.Context, db *database.APIServerDB, bootstrapApiURL, bootstrapApiCredentials string, publicKey []byte, apiEndpoint string) {
	parts := strings.Split(bootstrapApiCredentials, ":")
	bat := BasicAuthTransport{
		Username: parts[0],
		Password: parts[1],
	}

	bs := bootstrapper{
		Client: bat.Client(),
	}

	for {
		select {
		case <-time.After(1 * time.Second):
			deviceInfos, err := bs.fetchDeviceInfos(bootstrapApiURL)
			if err != nil {
				log.Errorf("bootstrap: Fetching device infos: %v", err)
				continue
			}

			for _, enrollment := range deviceInfos {
				err := db.AddDevice(ctx, database.Device{
					Username:  enrollment.Owner,
					PublicKey: enrollment.PublicKey,
					Serial:    enrollment.Serial,
					Platform:  enrollment.Platform,
				})

				if err != nil {
					log.Errorf("bootstrap: Adding device: %v", err)
					continue
				}

				device, err := db.ReadDevice(enrollment.PublicKey)
				if err != nil {
					log.Errorf("bootstrap: Getting device: %v", err)
					continue
				}

				bootstrapConfig := bootstrap.Config{
					DeviceIP:       device.IP,
					PublicKey:      string(publicKey),
					TunnelEndpoint: apiEndpoint,
					APIServerIP:    "10.255.240.1",
				}

				err = bs.pushBootstrapConfig(bootstrapApiURL, device.Serial, bootstrapConfig)

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

func (bs *bootstrapper) pushBootstrapConfig(bootstrapURL, serial string, bootstrapConfig bootstrap.Config) error {
	b, err := json.Marshal(bootstrapConfig)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	r, err := bs.Client.Post(fmt.Sprintf("%s/api/v2/device/config/%s", bootstrapURL, serial), "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("posting bootstrap config: %w", err)
	}

	log.Infof("bootstrap: Bootstrapped %+v, response: %v", bootstrapConfig, r)
	return nil
}

func (bs *bootstrapper) fetchDeviceInfos(bootstrapURL string) ([]bootstrap.DeviceInfo, error) {
	r, err := bs.Client.Get(bootstrapURL + "/api/v2/device/info")
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
