package database_test

import (
	"context"
	"github.com/nais/device/apiserver/database"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func setup(t *testing.T) *database.APIServerDB {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test")
	}

	db, err := database.NewTestDatabase("postgresql://postgres:postgres@localhost:5433", "../database/schema/schema.sql")
	if err != nil {
		t.Fatalf("Instantiating database: %v", err)
	}
	return db
}

func TestAPIServerDB_AddDevice(t *testing.T) {
	db := setup(t)

	ctx := context.Background()
	serial := "serial"
	d := database.Device{Username: "username", PublicKey: "publickey", Serial: serial, Platform: "darwin"}
	err := db.AddDevice(ctx, d)
	assert.NoError(t, err)

	device, err := db.ReadDevice(d.PublicKey)
	assert.NoError(t, err)
	assert.Equal(t, d.Username, device.Username)
	assert.Equal(t, d.PublicKey, device.PublicKey)
	assert.Equal(t, d.Serial, device.Serial)
	assert.Equal(t, d.Platform, device.Platform)
	assert.False(t, *device.Healthy)

	err = db.AddDevice(ctx, d)
	assert.NoError(t, err)

	newUsername, newPublicKey := "newUsername", "newPublicKey"
	dUpdated := database.Device{Username: newUsername, PublicKey: newPublicKey, Serial: serial, Platform: "darwin"}
	err = db.AddDevice(ctx, dUpdated)
	assert.NoError(t, err)

	device, err = db.ReadDevice(newPublicKey)
	assert.NoError(t, err)
	assert.Equal(t, dUpdated.Username, device.Username)
	assert.Equal(t, dUpdated.PublicKey, device.PublicKey)
}
