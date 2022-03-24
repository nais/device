//go:build integration_test
// +build integration_test

package database_test

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/testdatabase"
	"github.com/nais/device/pkg/pb"
)

const (
	timeout                 = 5 * time.Second
	wireguardNetworkAddress = "10.255.240.1/21"
	apiserverWireGuardIP    = "10.255.240.1"
)

func setup(t *testing.T) database.APIServer {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ipAllocator := database.NewIPAllocator(netip.MustParsePrefix(wireguardNetworkAddress), []string{apiserverWireGuardIP})
	db, err := testdatabase.New(ctx, "user=postgres password=postgres host=localhost port=5433 sslmode=disable", ipAllocator)
	if err != nil {
		t.Fatalf("Instantiating database: %v", err)
	}
	return db
}

func TestAddGateway(t *testing.T) {
	db := setup(t)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	g := pb.Gateway{
		Endpoint:     "1.2.3.4:56789",
		PublicKey:    "publicKey",
		Name:         "gateway",
		PasswordHash: "hunter2",
	}

	t.Run("adding new gateway works", func(t *testing.T) {
		err := db.AddGateway(ctx, &g)
		assert.NoError(t, err)

		gateway, err := db.ReadGateway(ctx, g.Name)
		assert.NoError(t, err)

		assert.Equal(t, g.Name, gateway.Name)
		assert.Equal(t, g.Endpoint, gateway.Endpoint)
		assert.Equal(t, g.PublicKey, gateway.PublicKey)
		assert.Equal(t, g.PasswordHash, gateway.PasswordHash)
		assert.False(t, gateway.RequiresPrivilegedAccess)
	})

	t.Run("adding a gateway with same name updates it", func(t *testing.T) {
		existingGateway, err := db.ReadGateway(ctx, g.Name)
		assert.NoError(t, err)
		existingGateway.PublicKey = "newpublickey"
		assert.NoError(t, db.AddGateway(ctx, existingGateway))
		resultingGateway, err := db.ReadGateway(ctx, g.Name)
		assert.NoError(t, err)
		assert.Equal(t, "newpublickey", resultingGateway.PublicKey)
	})

	t.Run("adding a gateway with an existing public key fails", func(t *testing.T) {
		existingGateway, err := db.ReadGateway(ctx, g.Name)
		assert.NoError(t, err)
		existingGateway.Name = "new name"
		assert.Error(t, db.AddGateway(ctx, existingGateway))
	})

	t.Run("updating existing gateway works", func(t *testing.T) {
		existingGateway, err := db.ReadGateway(ctx, g.Name)
		assert.NoError(t, err)

		assert.Nil(t, existingGateway.Routes)
		assert.Nil(t, existingGateway.AccessGroupIDs)

		existingGateway.Routes = []string{"r", "o", "u", "t", "e", "s"}
		existingGateway.AccessGroupIDs = []string{"a1", "b2", "c3"}
		existingGateway.RequiresPrivilegedAccess = true
		existingGateway.PublicKey = "new public key"
		existingGateway.Endpoint = "new endpoint"
		existingGateway.PasswordHash = "new password hash"

		assert.NoError(t, db.UpdateGateway(ctx, existingGateway))

		updatedGateway, err := db.ReadGateway(ctx, g.Name)
		assert.NoError(t, err)

		assert.Equal(t, existingGateway.Routes, updatedGateway.Routes)
		assert.Equal(t, existingGateway.AccessGroupIDs, updatedGateway.AccessGroupIDs)
		assert.True(t, updatedGateway.RequiresPrivilegedAccess)
		assert.Equal(t, existingGateway.PublicKey, updatedGateway.PublicKey)
		assert.Equal(t, existingGateway.Endpoint, updatedGateway.Endpoint)
		assert.Equal(t, existingGateway.PasswordHash, updatedGateway.PasswordHash)
	})
	t.Run("updating non-existent gateway is ok", func(t *testing.T) {
		nonExistentGateway := pb.Gateway{
			Name:           "non-existent",
			Routes:         []string{"r", "o", "u", "t", "e", "s"},
			AccessGroupIDs: []string{"a1", "b2", "c3"},
		}
		assert.NoError(t, db.UpdateGateway(ctx, &nonExistentGateway))
	})
}

func TestAddDevice(t *testing.T) {
	db := setup(t)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	serial := "serial"
	d := &pb.Device{Username: "username", PublicKey: "publickey", Serial: serial, Platform: "darwin"}
	err := db.AddDevice(ctx, d)
	assert.NoError(t, err)

	device, err := db.ReadDevice(ctx, d.PublicKey)
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

	device, err = db.ReadDevice(ctx, newPublicKey)
	assert.NoError(t, err)
	assert.Equal(t, dUpdated.Username, device.Username)
	assert.Equal(t, dUpdated.PublicKey, device.PublicKey)
}
