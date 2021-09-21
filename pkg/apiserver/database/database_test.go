// +build integration_test

package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/testdatabase"
	"github.com/nais/device/pkg/pb"
)

func setup(t *testing.T) database.APIServer {
	ctx := context.Background()
	db, err := testdatabase.New(ctx, "user=postgres password=postgres host=localhost port=5433 sslmode=disable")
	if err != nil {
		t.Fatalf("Instantiating database: %v", err)
	}
	return db
}

func TestAddGateway(t *testing.T) {
	db := setup(t)

	ctx := context.Background()
	g := pb.Gateway{
		Endpoint:  "1.2.3.4:56789",
		PublicKey: "publicKey",
		Name:      "gateway",
	}

	t.Run("adding new gateway works", func(t *testing.T) {
		err := db.AddGateway(ctx, g.Name, g.Endpoint, g.PublicKey)
		assert.NoError(t, err)

		gateway, err := db.ReadGateway(g.Name)
		assert.NoError(t, err)

		assert.Equal(t, g.Name, gateway.Name)
		assert.Equal(t, g.Endpoint, gateway.Endpoint)
		assert.Equal(t, g.PublicKey, gateway.PublicKey)
		assert.False(t, gateway.RequiresPrivilegedAccess)
	})

	t.Run("adding a gateway with same name as existing fails", func(t *testing.T) {
		existingGateway, err := db.ReadGateway(g.Name)
		assert.NoError(t, err)
		assert.Error(t, db.AddGateway(ctx, existingGateway.Name, existingGateway.Endpoint, existingGateway.PublicKey))
	})

	t.Run("updating existing gateway works", func(t *testing.T) {
		existingGateway, err := db.ReadGateway(g.Name)
		assert.NoError(t, err)

		assert.Nil(t, existingGateway.Routes)
		assert.Nil(t, existingGateway.AccessGroupIDs)

		routes := []string{"r", "o", "u", "t", "e", "s"}
		accessGroupIDs := []string{"a1", "b2", "c3"}

		assert.NoError(t, db.UpdateGateway(ctx, existingGateway.Name, routes, accessGroupIDs, true))

		updatedGateway, err := db.ReadGateway(g.Name)
		assert.NoError(t, err)

		assert.Equal(t, routes, updatedGateway.Routes)
		assert.Equal(t, accessGroupIDs, updatedGateway.AccessGroupIDs)
		assert.True(t, updatedGateway.RequiresPrivilegedAccess)
	})
	t.Run("updating non-existant gateway is ok", func(t *testing.T) {
		routes := []string{"r", "o", "u", "t", "e", "s"}
		accessGroupIDs := []string{"a1", "b2", "c3"}

		assert.NoError(t, db.UpdateGateway(ctx, "non-existant", routes, accessGroupIDs, false))
	})
}

func TestAddDevice(t *testing.T) {
	db := setup(t)

	ctx := context.Background()
	serial := "serial"
	d := &pb.Device{Username: "username", PublicKey: "publickey", Serial: serial, Platform: "darwin"}
	err := db.AddDevice(ctx, d)
	assert.NoError(t, err)

	device, err := db.ReadDevice(d.PublicKey)
	assert.NoError(t, err)
	assert.Equal(t, d.Username, device.Username)
	assert.Equal(t, d.PublicKey, device.PublicKey)
	assert.Equal(t, d.Serial, device.Serial)
	assert.Equal(t, d.Platform, device.Platform)
	assert.False(t, device.Healthy)

	err = db.AddDevice(ctx, d)
	assert.NoError(t, err)

	newUsername, newPublicKey := "newUsername", "newPublicKey"
	dUpdated := &pb.Device{Username: newUsername, PublicKey: newPublicKey, Serial: serial, Platform: "darwin"}
	err = db.AddDevice(ctx, dUpdated)
	assert.NoError(t, err)

	device, err = db.ReadDevice(newPublicKey)
	assert.NoError(t, err)
	assert.Equal(t, dUpdated.Username, device.Username)
	assert.Equal(t, dUpdated.PublicKey, device.PublicKey)
}
