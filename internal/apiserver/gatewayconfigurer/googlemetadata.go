package gatewayconfigurer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/ioconvenience"
	"github.com/sirupsen/logrus"
)

type GoogleMetadata struct {
	db  database.Database
	log logrus.FieldLogger
}

type GatewayMetadata struct {
	Routes                   []string `json:"routes"`
	AccessGroupIDs           []string `json:"access_group_ids"`
	RequiresPrivilegedAccess bool     `json:"requires_privileged_access"`
}

func NewGoogleMetadata(db database.Database, log logrus.FieldLogger) *GoogleMetadata {
	return &GoogleMetadata{
		db:  db,
		log: log,
	}
}

func (g *GoogleMetadata) SyncConfig(ctx context.Context) error {
	gatewayRoutes, err := getGatewayMetadatas(ctx, g.log)
	if err != nil {
		return err
	}
	for name, gatewayMetadata := range gatewayRoutes {
		gateway, err := g.db.ReadGateway(ctx, name)
		if err != nil {
			g.log.WithError(err).Error("read gateway")
			continue
		}

		gateway.RoutesIPv4 = gatewayMetadata.Routes
		gateway.AccessGroupIDs = gatewayMetadata.AccessGroupIDs
		gateway.RequiresPrivilegedAccess = gatewayMetadata.RequiresPrivilegedAccess

		err = g.db.UpdateGateway(ctx, gateway)
		if err != nil {
			g.log.WithError(err).Error("update gateway")
		}
	}
	return nil
}

func getGatewayMetadatas(ctx context.Context, log logrus.FieldLogger) (map[string]*GatewayMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/attributes/gateway-routes", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer ioconvenience.CloseWithLog(log, resp.Body)

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
