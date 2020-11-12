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

func (e *Enroller) WatchGatewayEnrollments(ctx context.Context) {
	for {
		select {
		case <-time.After(1 * time.Second):
			gatewayInfos, err := e.fetchGatewayInfos()
			if err != nil {
				log.Errorf("bootstrap: Fetching gateway infos: %v", err)
				continue
			}

			for _, enrollment := range gatewayInfos {
				err := e.DB.AddGateway(ctx, database.Gateway{
					PublicKey: enrollment.PublicKey,
					Name:      enrollment.Name,
					Endpoint:  enrollment.PublicIP,
				})

				if err != nil {
					log.Errorf("bootstrap: Adding gateway: %v", err)
					continue
				}

				gateway, err := e.DB.ReadGateway(enrollment.Name)
				if err != nil {
					log.Errorf("bootstrap: Getting gateway: %v", err)
					continue
				}

				bootstrapConfig := bootstrap.Config{
					DeviceIP:       gateway.IP,
					PublicKey:      e.APIServerPublicKey,
					TunnelEndpoint: e.APIServerEndpoint,
					APIServerIP:    "10.255.240.1",
				}

				err = e.postGatewayConfig(e.BootstrapAPIURL, gateway.Name, bootstrapConfig)

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

func (e *Enroller) postGatewayConfig(bootstrapURL, name string, bootstrapConfig bootstrap.Config) error {
	b, err := json.Marshal(bootstrapConfig)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}

	r, err := e.Client.Post(fmt.Sprintf("%s/api/v2/gateway/config/%s", bootstrapURL, name), "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("posting bootstrap config: %w", err)
	}

	log.Infof("bootstrap: Bootstrapped %+v, response: %v", bootstrapConfig, r)
	return nil
}

func (e *Enroller) fetchGatewayInfos() ([]bootstrap.GatewayInfo, error) {
	r, err := e.Client.Get(e.BootstrapAPIURL + "/api/v2/gateway/info")
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
