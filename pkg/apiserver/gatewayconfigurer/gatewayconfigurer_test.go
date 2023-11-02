package gatewayconfigurer_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nais/device/pkg/apiserver/bucket"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nais/device/pkg/apiserver/gatewayconfigurer"
)

const (
	gatewayName, route, accessGroupId = "name", "r", "agid"
	requiresPrivilegedAccess          = true
)

var errExpected = errors.New("expected error")

func TestGatewayConfigurer_SyncConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	log := logrus.StandardLogger().WithField("component", "test")

	t.Run("updates gateway config in database according to bucket definition", func(t *testing.T) {
		db := &database.MockAPIServer{}
		mockClient := &bucket.MockClient{}
		mockObject := &bucket.MockObject{}
		lastUpdated := time.Now()
		reader := strings.NewReader(gatewayConfig(gatewayName, route, accessGroupId, requiresPrivilegedAccess))

		gc := gatewayconfigurer.NewGatewayConfigurer(log, db, mockClient, 0)

		db.On("UpdateGatewayDynamicFields",
			mock.Anything,
			&pb.Gateway{
				Name:                     gatewayName,
				RoutesIPv4:               []string{route},
				AccessGroupIDs:           []string{accessGroupId},
				RequiresPrivilegedAccess: requiresPrivilegedAccess,
			},
		).Return(nil).Once()

		mockClient.On("Open", mock.Anything).Return(mockObject, nil).Twice()
		mockObject.On("LastUpdated").Return(lastUpdated).Twice()
		mockObject.On("Close").Return(nil).Twice()
		mockObject.On("Reader").Return(reader).Once()

		err := gc.SyncConfig(ctx)

		assert.NoError(t, err)

		err = gc.SyncConfig(ctx)
		assert.NoError(t, err)

		mock.AssertExpectationsForObjects(t, db, mockClient, mockObject)
	})

	t.Run("handles errors from bucket interface", func(t *testing.T) {
		db := &database.MockAPIServer{}
		mockClient := &bucket.MockClient{}
		mockObject := &bucket.MockObject{}

		gc := gatewayconfigurer.NewGatewayConfigurer(log, db, mockClient, 0)

		mockClient.On("Open", mock.Anything).Return(nil, errExpected).Once()

		err := gc.SyncConfig(ctx)

		assert.EqualError(t, err, "open bucket: expected error")
		mock.AssertExpectationsForObjects(t, mockClient, mockObject)
	})

	t.Run("handles errors from unmarshal", func(t *testing.T) {
		db := &database.MockAPIServer{}
		mockClient := &bucket.MockClient{}
		mockObject := &bucket.MockObject{}
		lastUpdated := time.Now()
		reader := strings.NewReader(`this is not valid json`)

		gc := gatewayconfigurer.NewGatewayConfigurer(log, db, mockClient, 0)

		mockClient.On("Open", mock.Anything).Return(mockObject, nil).Once()
		mockObject.On("LastUpdated").Return(lastUpdated).Once()
		mockObject.On("Close").Return(nil).Once()
		mockObject.On("Reader").Return(reader).Once()

		err := gc.SyncConfig(ctx)

		assert.Error(t, err)

		mock.AssertExpectationsForObjects(t, mockClient, mockObject)
	})

	t.Run("handles errors from updategateway", func(t *testing.T) {
		db := &database.MockAPIServer{}
		mockClient := &bucket.MockClient{}
		mockObject := &bucket.MockObject{}
		lastUpdated := time.Now()
		reader := strings.NewReader(gatewayConfig(gatewayName, route, accessGroupId, requiresPrivilegedAccess))

		gc := gatewayconfigurer.NewGatewayConfigurer(log, db, mockClient, 0)

		db.On("UpdateGatewayDynamicFields",
			mock.Anything,
			mock.Anything,
		).Return(errExpected).Once()
		mockClient.On("Open", mock.Anything).Return(mockObject, nil).Once()
		mockObject.On("LastUpdated").Return(lastUpdated).Once()
		mockObject.On("Close").Return(nil).Once()
		mockObject.On("Reader").Return(reader).Once()

		err := gc.SyncConfig(ctx)

		assert.Error(t, err)

		mock.AssertExpectationsForObjects(t, db, mockClient, mockObject)
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
