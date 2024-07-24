package database_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/nais/device/internal/apiserver/testdatabase"
	"github.com/nais/device/internal/pb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const timeout = time.Second * 5

func TestAddGateway(t *testing.T) {
	db := testdatabase.Setup(t, false)

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
	db := testdatabase.Setup(t, true)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	serial := "serial"
	d := &pb.Device{Username: "username", PublicKey: "publickey", Serial: serial, Platform: "darwin", LastSeen: timestamppb.Now(), ExternalID: "1"}
	checks := []*kolide.Check{
		{
			ID:          1,
			Tags:        []string{"critical"},
			DisplayName: "check display name",
			Description: "check description",
		},
	}
	externalID, err := strconv.Atoi(d.ExternalID)
	if err != nil {
		t.Fatal(err)
	}
	issues := []*kolide.DeviceFailure{
		{
			Title: "integration test issue",
			Device: kolide.Device{
				ID: int64(externalID),
			},
			CheckID:   checks[0].ID,
			Timestamp: &time.Time{},
		},
	}
	err = db.AddDevice(ctx, d)
	assert.NoError(t, err)

	err = db.UpdateKolideChecks(ctx, checks)
	assert.NoError(t, err)

	err = db.UpdateKolideIssuesForDevice(ctx, d.ExternalID, issues)
	assert.NoError(t, err)

	ls := d.LastSeen.AsTime()
	err = db.SetDeviceSeenByKolide(ctx, d.ExternalID, d.Serial, d.Platform, &ls)
	assert.NoError(t, err)

	device, err := db.ReadDevice(ctx, d.PublicKey)
	assert.NoError(t, err)

	assert.Equal(t, d.Username, device.Username)
	assert.Equal(t, d.PublicKey, device.PublicKey)
	assert.Equal(t, d.Serial, device.Serial)
	assert.Equal(t, d.Platform, device.Platform)

	assert.Len(t, device.Issues, 1)
	assert.Equal(t, issues[0].Title, device.Issues[0].Title)
	assert.Equal(t, fmt.Sprint(issues[0].Device.ID), device.ExternalID)

	// re-add also works
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

func TestReadPeers(t *testing.T) {
	db := testdatabase.Setup(t, true)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	d1 := &pb.Device{
		Username:  "user1@example.com",
		PublicKey: "publickey1",
		Serial:    "serial1",
		Platform:  "darwin",
		LastSeen:  timestamppb.Now(),
	}
	err := db.AddDevice(ctx, d1)
	assert.NoError(t, err)

	d2 := &pb.Device{
		Username:  "user2@example.com",
		PublicKey: "publickey2",
		Serial:    "serial2",
		Platform:  "darwin",
		LastSeen:  timestamppb.Now(),
	}
	err = db.AddDevice(ctx, d2)
	assert.NoError(t, err)

	dbdevice1, err := db.ReadDeviceBySerialPlatform(ctx, d1.Serial, d1.Platform)
	assert.NoError(t, err)
	dbdevice2, err := db.ReadDeviceBySerialPlatform(ctx, d2.Serial, d2.Platform)
	assert.NoError(t, err)

	peers, err := db.ReadPeers(ctx)
	assert.NoError(t, err)

	assert.Equal(t, d1.Username, peers[0].GetName())
	assert.Equal(t, dbdevice1.Ipv4+"/32", peers[0].GetAllowedIPs()[0])
	assert.Equal(t, d1.PublicKey, peers[0].GetPublicKey())

	assert.Equal(t, d2.Username, peers[1].GetName())
	assert.Equal(t, dbdevice2.Ipv4+"/32", peers[1].GetAllowedIPs()[0])
	assert.Equal(t, d2.PublicKey, peers[1].GetPublicKey())
}
