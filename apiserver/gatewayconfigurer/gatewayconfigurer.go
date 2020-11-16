package gatewayconfigurer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
	"io"
	"strings"
	"time"
)

type BucketReader interface {
	ReadBucketObject() (io.Reader, error)
}

type GatewayConfigurer struct {
	DB           *database.APIServerDB
	BucketReader BucketReader
	SyncInterval time.Duration
}

type GatewayConfig struct {
	Routes         string `json:"routes"`
	AccessGroupIds string `json:"access_group_ids"`
}

func (g *GatewayConfigurer) SyncContinuously(ctx context.Context) {
	for {
		select {
		case <-time.After(g.SyncInterval):
			if err := g.SyncConfig(); err != nil {
				log.Errorf("Synchronizing gateway configuration: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (g *GatewayConfigurer) SyncConfig() error {
	reader, err := g.BucketReader.ReadBucketObject()
	if err != nil {
		return fmt.Errorf("reading bucket object: %v", err)
	}

	var gatewayConfigs map[string]GatewayConfig
	if err := json.NewDecoder(reader).Decode(&gatewayConfigs); err != nil {
		return fmt.Errorf("unmarshaling gateway config json: %v", err)
	}

	for gatewayName, gatewayConfig := range gatewayConfigs {
		routes := strings.Split(gatewayConfig.Routes, ",")
		accessGroupIds := strings.Split(gatewayConfig.AccessGroupIds, ",")
		if err := g.DB.UpdateGateway(context.Background(), gatewayName, routes, accessGroupIds); err != nil {
			return fmt.Errorf("updating gateway: %s with routes: %s and accessGroupIds: %s: %v", gatewayName, routes, accessGroupIds, err)
		}
	}

	return nil
}
