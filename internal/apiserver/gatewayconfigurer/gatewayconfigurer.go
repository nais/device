package gatewayconfigurer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nais/device/internal/apiserver/bucket"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/ioconvenience"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
)

type GatewayConfigurer struct {
	db          database.Database
	bucket      bucket.Client
	lastUpdated time.Time
	log         *logrus.Entry
}

func NewGatewayConfigurer(log *logrus.Entry, db database.Database, bucket bucket.Client) *GatewayConfigurer {
	return &GatewayConfigurer{
		db:     db,
		bucket: bucket,
		log:    log,
	}
}

type Route struct {
	CIDR string `json:"cidr"`
}

type GatewayConfig struct {
	Routes                   []Route  `json:"routes"`
	RoutesIPv6               []Route  `json:"routes_ipv6"`
	AccessGroupIds           []string `json:"access_group_ids"`
	RequiresPrivilegedAccess bool     `json:"requires_privileged_access"`
}

func (g *GatewayConfigurer) SyncConfig(ctx context.Context) error {
	object, err := g.bucket.Open(ctx)
	if err != nil {
		return fmt.Errorf("open bucket: %w", err)
	}
	defer ioconvenience.CloseWithLog(g.log, object)

	// only update configuration if changed server-side
	lastUpdated := object.LastUpdated()
	if g.lastUpdated.Equal(lastUpdated) {
		return nil
	}

	g.log.Info("syncing gateway configuration from bucket")
	var gatewayConfigs map[string]GatewayConfig
	if err := json.NewDecoder(object.Reader()).Decode(&gatewayConfigs); err != nil {
		return fmt.Errorf("unmarshaling gateway config json: %v", err)
	}

	for gatewayName, gatewayConfig := range gatewayConfigs {
		gw := &pb.Gateway{
			Name:                     gatewayName,
			AccessGroupIDs:           gatewayConfig.AccessGroupIds,
			RequiresPrivilegedAccess: gatewayConfig.RequiresPrivilegedAccess,
			RoutesIPv4:               ToCIDRStringSlice(gatewayConfig.Routes),
			RoutesIPv6:               ToCIDRStringSlice(gatewayConfig.RoutesIPv6),
		}

		err = g.db.UpdateGatewayDynamicFields(ctx, gw)
		if err != nil {
			return fmt.Errorf("updating gateway: %s with routes: %s and accessGroupIds: %s: %w", gatewayName, gatewayConfig.Routes, gatewayConfig.AccessGroupIds, err)
		}
	}

	g.lastUpdated = lastUpdated

	return nil
}

func ToCIDRStringSlice(routeObjects []Route) []string {
	var routes []string
	for _, route := range routeObjects {
		routes = append(routes, route.CIDR)
	}

	return routes
}
