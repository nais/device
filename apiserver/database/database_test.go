package database_test

import (
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

	username, publicKey, serial, platform := "username", "publicKey", "serial", "darwin"

	err := db.AddDevice(username, publicKey, serial, platform)
	assert.NoError(t, err)

	device, err := db.ReadDevice(publicKey)
	assert.NoError(t, err)
	assert.Equal(t, username, device.Username)
	assert.Equal(t, publicKey, device.PublicKey)
	assert.Equal(t, serial, device.Serial)
	assert.Equal(t, platform, device.Platform)
	assert.False(t, *device.Healthy)

	err = db.AddDevice(username, publicKey, serial, platform)
	assert.NoError(t, err)

	newUsername, newPublicKey := "newUsername", "newPublicKey"
	err = db.AddDevice(newUsername, newPublicKey, serial, platform)
	assert.NoError(t, err)

	device, err = db.ReadDevice(newPublicKey)
	assert.NoError(t, err)
	assert.Equal(t, newUsername, device.Username)
	assert.Equal(t, newPublicKey, device.PublicKey)
}
