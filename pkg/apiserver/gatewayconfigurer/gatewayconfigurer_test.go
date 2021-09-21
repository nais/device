// +build integration_test

package gatewayconfigurer_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nais/device/pkg/apiserver/gatewayconfigurer"
	"github.com/nais/device/pkg/apiserver/testdatabase"
)

func TestGatewayConfigurer_SyncConfig(t *testing.T) {
	t.Run("updates gateway config in database according to bucket definition", func(t *testing.T) {
		ctx := context.Background()
		testDB, err := testdatabase.New(ctx, "user=postgres password=postgres host=localhost port=5433 sslmode=disable")
		assert.NoError(t, err)
		const gatewayName, route, accessGroupId = "name", "r", "agid"
		assert.NoError(t, testDB.AddGateway(context.Background(), gatewayName, "", ""))

		bucketReader := MockBucketReader{GatewayConfigs: gatewayConfig(gatewayName, route, accessGroupId, true)}

		gc := gatewayconfigurer.GatewayConfigurer{
			DB:           testDB,
			BucketReader: bucketReader,
		}

		gateway, err := testDB.ReadGateway(gatewayName)
		assert.NoError(t, err)
		assert.Equal(t, gatewayName, gateway.Name)
		assert.Nil(t, gateway.Routes)
		assert.Nil(t, gateway.AccessGroupIDs)
		assert.False(t, gateway.RequiresPrivilegedAccess)

		assert.NoError(t, gc.SyncConfig(context.Background()))

		updatedGateway, err := testDB.ReadGateway(gatewayName)
		assert.NoError(t, err)
		assert.Len(t, updatedGateway.Routes, 1)
		assert.Equal(t, route, updatedGateway.Routes[0])
		assert.Len(t, updatedGateway.AccessGroupIDs, 1)
		assert.Equal(t, accessGroupId, updatedGateway.AccessGroupIDs[0])
		assert.True(t, updatedGateway.RequiresPrivilegedAccess)
	})

	t.Run("synchronizing gatewayconfig where gateway not in database is ok", func(t *testing.T) {
		ctx := context.Background()
		testDB, err := testdatabase.New(ctx, "user=postgres password=postgres host=localhost port=5433 sslmode=disable")

		assert.NoError(t, err)
		const gatewayName, route, accessGroupId = "name", "r", "agid"

		bucketReader := MockBucketReader{GatewayConfigs: gatewayConfig(gatewayName, route, accessGroupId, true)}

		gc := gatewayconfigurer.GatewayConfigurer{
			DB:           testDB,
			BucketReader: bucketReader,
		}

		gw, err := testDB.ReadGateway(gatewayName)
		assert.Error(t, err)
		assert.Nil(t, gw)

		assert.NoError(t, gc.SyncConfig(context.Background()))
	})

}

func gatewayConfig(gatewayName string, route string, accessGroupId string, requiresPrivilegedAccess bool) string {
	gatewayConfigs := fmt.Sprintf(
		`{
				"%s": {
					"routes": [{"cidr": "%s"}],
					"access_group_ids": ["%s"],
					"requires_privileged_access": %t
				}
			 }`, gatewayName, route, accessGroupId, requiresPrivilegedAccess)
	return gatewayConfigs
}

type MockBucketReader struct {
	GatewayConfigs string
}

func (m MockBucketReader) ReadBucketObject(_ context.Context) (io.Reader, error) {
	return strings.NewReader(m.GatewayConfigs), nil
}
