package gatewayconfigurer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nais/device/pkg/apiserver/database"
	log "github.com/sirupsen/logrus"
)

type GoogleMetadata struct {
	db  database.APIServer
	log *log.Entry
}

type GatewayMetadata struct {
	Routes                   []string `json:"routes"`
	AccessGroupIDs           []string `json:"access_group_ids"`
	RequiresPrivilegedAccess bool     `json:"requires_privileged_access"`
}

func NewGoogleMetadata(db database.APIServer, log *log.Entry) *GoogleMetadata {
	return &GoogleMetadata{
		db:  db,
		log: log,
	}
}

func (g *GoogleMetadata) SyncContinuously(ctx context.Context, syncInterval time.Duration) {
	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := g.syncConfig(ctx); err != nil {
				g.log.WithError(err).Error("Synchronizing gateway configuration")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (g *GoogleMetadata) syncConfig(ctx context.Context) error {
	gatewayRoutes, err := getGatewayMetadatas(ctx)
	if err != nil {
		return err
	}
	for name, gatewayMetadata := range gatewayRoutes {
		gateway, err := g.db.ReadGateway(ctx, name)
		if err != nil {
			g.log.WithError(err).Error("read gateway")
			continue
		}

		gateway.Routes = gatewayMetadata.Routes
		gateway.AccessGroupIDs = gatewayMetadata.AccessGroupIDs
		gateway.RequiresPrivilegedAccess = gatewayMetadata.RequiresPrivilegedAccess

		err = g.db.UpdateGateway(ctx, gateway)
		if err != nil {
			g.log.WithError(err).Error("update gateway")
		}
	}
	return nil
}

func getGatewayMetadatas(ctx context.Context) (map[string]*GatewayMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/attributes/gateway-routes", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status on metadata request: %v", resp.Status)
	}

	var gatewayMetadatas map[string]*GatewayMetadata
	err = json.NewDecoder(resp.Body).Decode(&gatewayMetadatas)
	if err != nil {
		return nil, err
	}

	return gatewayMetadatas, nil
}
