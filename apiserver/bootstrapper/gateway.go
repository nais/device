package bootstrapper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/pkg/basicauth"
	"github.com/nais/device/pkg/bootstrap"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

func WatchGatewayEnrollments(ctx context.Context, db *database.APIServerDB, bootstrapBaseApiUrl, bootstrapApiCredentials string, publicKey []byte, apiEndpoint string) {
	parts := strings.Split(bootstrapApiCredentials, ":")
	bat := &basicauth.Transport{
		Username: parts[0],
		Password: parts[1],
	}

	bs := bootstrapper{
		Client: bat.Client(),
	}

	for {
		select {
		case <-time.After(1 * time.Second):
			gatewayInfos, err := bs.fetchGatewayInfos(bootstrapBaseApiUrl)
			if err != nil {
				log.Errorf("bootstrap: Fetching gateway infos: %v", err)
				continue
			}

			for _, enrollment := range gatewayInfos {
				err := db.AddGateway(ctx, database.Gateway{
					PublicKey: enrollment.PublicKey,
				})

				if err != nil {
					log.Errorf("bootstrap: Adding gateway: %v", err)
					continue
				}

				gateway, err := db.ReadGateway(enrollment.PublicKey)
				if err != nil {
					log.Errorf("bootstrap: Getting gateway: %v", err)
					continue
				}

				bootstrapConfig := bootstrap.Config{
					DeviceIP:       "x.x.x.x",
					PublicKey:      string(publicKey),
					TunnelEndpoint: apiEndpoint,
					APIServerIP:    "10.255.240.1",
				}

				err = bs.postGatewayConfig(bootstrapBaseApiUrl, gateway.Name, bootstrapConfig)

				if err != nil {
					log.Errorf("bootstrap: Pushing bootstrap config: %v", err)
					continue
				}

				log.Infof("bootstrap: Bootstrapped gateway: %+v", bootstrapConfig)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (bs *bootstrapper) postGatewayConfig(bootstrapURL, serial string, bootstrapConfig bootstrap.Config) error {
	b, err := json.Marshal(bootstrapConfig)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	r, err := bs.Client.Post(fmt.Sprintf("%s/api/v2/gateway/config/%s", bootstrapURL, serial), "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("posting bootstrap config: %w", err)
	}

	log.Infof("bootstrap: Bootstrapped %+v, response: %v", bootstrapConfig, r)
	return nil
}

func (bs *bootstrapper) fetchGatewayInfos(bootstrapURL string) ([]bootstrap.GatewayInfo, error) {
	r, err := bs.Client.Get(bootstrapURL + "/api/v2/gateway/info")
	if err != nil {
		return nil, fmt.Errorf("getting gateway infos: %w", err)
	}

	var gatewayInfos []bootstrap.GatewayInfo
	err = json.NewDecoder(r.Body).Decode(&gatewayInfos)
	if err != nil {
		return nil, fmt.Errorf("decoding gatewayInfos: %w", err)
	}

	return gatewayInfos, nil
}
