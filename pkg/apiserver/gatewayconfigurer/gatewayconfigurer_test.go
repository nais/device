package gatewayconfigurer_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nais/device/pkg/apiserver/bucket"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nais/device/pkg/apiserver/gatewayconfigurer"
)

func TestGatewayConfigurer_SyncConfig(t *testing.T) {
	t.Run("updates gateway config in database according to bucket definition", func(t *testing.T) {
		ctx := context.Background()

		const gatewayName, route, accessGroupId = "name", "r", "agid"
		const requiresPrivilegedAccess = true

		//bucketReader := MockBucketReader{GatewayConfigs: gatewayConfig(gatewayName, route, accessGroupId, true)}
		channel := make(chan struct{}, 2)
		db := &database.MockAPIServer{}
		mockClient := &bucket.MockClient{}
		mockObject := &bucket.MockObject{}
		lastUpdated := time.Now()
		reader := strings.NewReader(gatewayConfig(gatewayName, route, accessGroupId, requiresPrivilegedAccess))

		gc := gatewayconfigurer.GatewayConfigurer{
			DB:                 db,
			Bucket:             mockClient,
			TriggerGatewaySync: channel,
		}

		db.On("UpdateGateway",
			mock.Anything,
			gatewayName,
			[]string{route},
			[]string{accessGroupId},
			requiresPrivilegedAccess,
		).Return(nil).Once()

		mockClient.On("Open", mock.Anything).Return(mockObject, nil).Once()
		mockObject.On("LastUpdated").Return(lastUpdated).Once()
		mockObject.On("Reader").Return(reader).Once()
		mockObject.On("Close").Return(nil).Once()

		err := gc.SyncConfig(ctx)

		assert.NoError(t, err)
		assert.Len(t, channel, 1)
		mock.AssertExpectationsForObjects(t, mockClient, mockObject)
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
