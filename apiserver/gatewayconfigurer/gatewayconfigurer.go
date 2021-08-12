package gatewayconfigurer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
)

type BucketReader interface {
	ReadBucketObject(ctx context.Context) (io.Reader, error)
}

type GatewayConfigurer struct {
	DB           *database.APIServerDB
	BucketReader BucketReader
	SyncInterval time.Duration
}

type Route struct {
	CIDR string `json:"cidr"`
}

type GatewayConfig struct {
	Routes                   []Route  `json:"routes"`
	AccessGroupIds           []string `json:"access_group_ids"`
	RequiresPrivilegedAccess bool     `json:"requires_privileged_access"`
}

func (g *GatewayConfigurer) SyncContinuously(ctx context.Context) {
	for {
		select {
		case <-time.After(g.SyncInterval):
			if err := g.SyncConfig(ctx); err != nil {
				log.Errorf("Synchronizing gateway configuration: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (g *GatewayConfigurer) SyncConfig(ctx context.Context) error {
	reader, err := g.BucketReader.ReadBucketObject(ctx)
	if err != nil {
		return fmt.Errorf("reading bucket object: %v", err)
	}

	var gatewayConfigs map[string]GatewayConfig
	if err := json.NewDecoder(reader).Decode(&gatewayConfigs); err != nil {
		return fmt.Errorf("unmarshaling gateway config json: %v", err)
	}

	for gatewayName, gatewayConfig := range gatewayConfigs {
		if err := g.DB.UpdateGateway(context.Background(), gatewayName, ToCIDRStringSlice(gatewayConfig.Routes), gatewayConfig.AccessGroupIds, gatewayConfig.RequiresPrivilegedAccess); err != nil {
			return fmt.Errorf("updating gateway: %s with routes: %s and accessGroupIds: %s: %v", gatewayName, gatewayConfig.Routes, gatewayConfig.AccessGroupIds, err)
		}
	}

	return nil
}

func ToCIDRStringSlice(routeObjects []Route) []string {
	var routes []string
	for _, route := range routeObjects {
		routes = append(routes, route.CIDR)
	}

	return routes
}
