package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/testdatabase"
	"github.com/nais/device/internal/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const timeout = time.Second * 5

func TestAddGateway(t *testing.T) {
	db := testdatabase.Setup(t)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	g := pb.Gateway{
		Endpoint:     "1.2.3.4:56789",
		PublicKey:    "publicKey",
		Name:         "gateway",
		PasswordHash: "hunter2",
		RoutesIPv4:   []string{"1.2.3.4/32"},
		RoutesIPv6:   []string{"fb32::1/128"},
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
		assert.Equal(t, g.GetRoutesIPv4(), gateway.GetRoutesIPv4())
		assert.Equal(t, g.GetRoutesIPv6(), gateway.GetRoutesIPv6())
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

		assert.Equal(t, g.GetRoutesIPv4(), existingGateway.GetRoutesIPv4())
		assert.Equal(t, g.GetRoutesIPv6(), existingGateway.GetRoutesIPv6())
		assert.Empty(t, existingGateway.AccessGroupIDs)

		existingGateway.RoutesIPv4 = []string{"e", "o", "r", "s", "t", "u"}
		existingGateway.AccessGroupIDs = []string{"a1", "b2", "c3"}
		existingGateway.RequiresPrivilegedAccess = true
		existingGateway.PublicKey = "new public key"
		existingGateway.Endpoint = "new endpoint"
		existingGateway.PasswordHash = "new password hash"

		assert.NoError(t, db.UpdateGateway(ctx, existingGateway))

		updatedGateway, err := db.ReadGateway(ctx, g.Name)
		assert.NoError(t, err)

		assert.Equal(t, existingGateway.GetRoutesIPv4(), updatedGateway.GetRoutesIPv4())
		assert.Equal(t, existingGateway.AccessGroupIDs, updatedGateway.AccessGroupIDs)
		assert.True(t, updatedGateway.RequiresPrivilegedAccess)
		assert.Equal(t, existingGateway.PublicKey, updatedGateway.PublicKey)
		assert.Equal(t, existingGateway.Endpoint, updatedGateway.Endpoint)
		assert.Equal(t, existingGateway.PasswordHash, updatedGateway.PasswordHash)
	})
}

func TestAddDevice(t *testing.T) {
	db := testdatabase.Setup(t)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	serial := "serial"
	issues := []*pb.DeviceIssue{
		{
			Title: "integration test issue",
		},
	}
	d := &pb.Device{Username: "username", PublicKey: "publickey", Serial: serial, Platform: "darwin", LastSeen: timestamppb.Now()}
	err := db.AddDevice(ctx, d)
	assert.NoError(t, err)

	ls := d.LastSeen.AsTime()
	err = db.UpdateSingleDevice(ctx, d.ExternalID, d.Serial, d.Platform, &ls, issues)
	assert.NoError(t, err)

	device, err := db.ReadDevice(ctx, d.PublicKey)
	assert.NoError(t, err)
	assert.Equal(t, d.Username, device.Username)
	assert.Equal(t, d.PublicKey, device.PublicKey)
	assert.Equal(t, d.Serial, device.Serial)
	assert.Equal(t, d.Platform, device.Platform)
	assert.EqualValues(t, issues, device.Issues)

	err = db.AddDevice(ctx, d)
	assert.NoError(t, err)

	newUsername, newPublicKey := "newUsername", "newPublicKey"
	dUpdated := &pb.Device{Username: newUsername, PublicKey: newPublicKey, Serial: serial, Platform: "darwin", LastSeen: timestamppb.Now()}
	err = db.AddDevice(ctx, dUpdated)
	assert.NoError(t, err)

	device, err = db.ReadDevice(ctx, newPublicKey)
	assert.NoError(t, err)
	assert.Equal(t, dUpdated.Username, device.Username)
	assert.Equal(t, dUpdated.PublicKey, device.PublicKey)
}
