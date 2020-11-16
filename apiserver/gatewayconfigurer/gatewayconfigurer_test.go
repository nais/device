package gatewayconfigurer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/apiserver/gatewayconfigurer"
	"github.com/nais/device/apiserver/testdatabase"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"strings"
	"testing"
)

func TestGatewayConfigurer_SyncConfig(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test")
	}

	t.Run("updates gateway config in database according to bucket definition", func(t *testing.T) {
		testDB, err := testdatabase.New("user=postgres password=postgres host=localhost port=5433 sslmode=disable", "../database/schema/schema.sql")
		assert.NoError(t, err)
		const gatewayName, routes, accessGroupIds = "name", "r,o,u,t,e,s", "a,g,i"
		assert.NoError(t, testDB.AddGateway(context.Background(), database.Gateway{
			Name: gatewayName,
		}))

		bucketReader := MockBucketReader{GatewayConfigs: &map[string]gatewayconfigurer.GatewayConfig{
			gatewayName: {
				Routes:         routes,
				AccessGroupIds: accessGroupIds,
			}}}

		gc := gatewayconfigurer.GatewayConfigurer{
			DB:           testDB,
			BucketReader: bucketReader,
		}

		gateway, err := testDB.ReadGateway(gatewayName)
		assert.NoError(t, err)
		assert.Equal(t, gatewayName, gateway.Name)
		assert.Nil(t, gateway.Routes)
		assert.Nil(t, gateway.AccessGroupIDs)

		assert.NoError(t, gc.SyncConfig())

		updatedGateway, err := testDB.ReadGateway(gatewayName)
		assert.NoError(t, err)
		assert.Equal(t, strings.Split(routes, ","), updatedGateway.Routes)
		assert.Equal(t, strings.Split(accessGroupIds, ","), updatedGateway.AccessGroupIDs)
	})

	t.Run("synchronizing gatewayconfig where gateway not in database is ok", func(t *testing.T) {
		testDB, err := testdatabase.New("user=postgres password=postgres host=localhost port=5433 sslmode=disable", "../database/schema/schema.sql")
		assert.NoError(t, err)
		const gatewayName, routes, accessGroupIds = "name", "r,o,u,t,e,s", "a,g,i"

		bucketReader := MockBucketReader{GatewayConfigs: &map[string]gatewayconfigurer.GatewayConfig{
			gatewayName: {
				Routes:         routes,
				AccessGroupIds: accessGroupIds,
			}}}

		gc := gatewayconfigurer.GatewayConfigurer{
			DB:           testDB,
			BucketReader: bucketReader,
		}

		gw, err := testDB.ReadGateway(gatewayName)
		assert.Error(t, err)
		assert.Nil(t, gw)

		assert.NoError(t, gc.SyncConfig())
	})

}

type MockBucketReader struct {
	GatewayConfigs *map[string]gatewayconfigurer.GatewayConfig
}

func (m MockBucketReader) ReadBucketObject() (io.Reader, error) {
	b, err := json.Marshal(m.GatewayConfigs)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(b), nil
}
