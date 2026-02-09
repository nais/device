package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/api"
	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestUpdateAllDevices_ClearsExternalIDWhenDeviceRemovedFromKolide tests that
// when a device is removed from Kolide (or no longer matches), its external_id
// should be cleared from the database.
func TestUpdateAllDevices_ClearsExternalIDWhenDeviceRemovedFromKolide(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup: We have 3 devices in our database, all with external_ids
	device1 := &pb.Device{
		Id:         1,
		Serial:     "serial1",
		Platform:   "darwin",
		Username:   "user1@example.com",
		PublicKey:  "publicKey1",
		ExternalID: "kolide-id-1", // This device is in Kolide
	}

	device2 := &pb.Device{
		Id:         2,
		Serial:     "serial2",
		Platform:   "linux",
		Username:   "user2@example.com",
		PublicKey:  "publicKey2",
		ExternalID: "kolide-id-2", // This device was REMOVED from Kolide
	}

	device3 := &pb.Device{
		Id:         3,
		Serial:     "serial3",
		Platform:   "windows",
		Username:   "user3@example.com",
		PublicKey:  "publicKey3",
		ExternalID: "kolide-id-3", // This device is in Kolide
	}

	// Kolide only returns 2 devices (device2 was removed/unmatched)
	kolideDevices := []*kolide.Device{
		{
			ID:       "kolide-id-1",
			Serial:   "serial1",
			Platform: "darwin",
		},
		{
			ID:       "kolide-id-3",
			Serial:   "serial3",
			Platform: "windows",
		},
	}

	db := database.NewMockDatabase(t)

	// First ReadDevices call returns all 3 devices with their external_ids
	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device1, device2, device3}, nil).Once()

	// UpdateKolideIssues will be called
	db.EXPECT().UpdateKolideIssues(mock.Anything, mock.Anything).Return(nil).Once()

	// UpdateDevices will be called with the devices
	var capturedDevices []*pb.Device
	db.EXPECT().UpdateDevices(mock.Anything, mock.MatchedBy(func(devices []*pb.Device) bool {
		// Capture the devices being updated so we can verify them
		capturedDevices = devices
		return true
	})).Return(nil).Once()

	// Second ReadDevices call after update
	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device1, device2, device3}, nil).Once()

	// Create a mock Kolide client that returns only 2 devices
	kolideClient := &mockKolideClient{
		devices: kolideDevices,
		issues:  []*kolide.Issue{},
	}

	sessionStore := auth.NewMockSessionStore(t)
	sessionStore.EXPECT().RefreshDevice(mock.Anything).Return().Maybe()

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, nil, nil, sessionStore, kolideClient, true)

	// Execute UpdateAllDevices
	err := server.UpdateAllDevices(ctx)
	assert.NoError(t, err)

	// Verify that the captured devices have the correct external_ids
	// device1 should keep its external_id (it's in Kolide)
	assert.Equal(t, "kolide-id-1", capturedDevices[0].ExternalID, "device1 should retain external_id")

	// device2 should have its external_id CLEARED (it's not in Kolide)
	// THIS ASSERTION WILL FAIL with the current implementation
	assert.Empty(t, capturedDevices[1].ExternalID, "device2 external_id should be cleared when not found in Kolide")

	// device3 should keep its external_id (it's in Kolide)
	assert.Equal(t, "kolide-id-3", capturedDevices[2].ExternalID, "device3 should retain external_id")
}

// TestUpdateAllDevices_ClearsExternalIDWhenSerialChanges tests that when a device's
// serial number changes locally, the external_id should be cleared until a new match is found
func TestUpdateAllDevices_ClearsExternalIDWhenSerialChanges(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup: Device has external_id but serial number has changed
	device := &pb.Device{
		Id:         1,
		Serial:     "new-serial", // Serial changed locally
		Platform:   "darwin",
		Username:   "user@example.com",
		PublicKey:  "publicKey",
		ExternalID: "kolide-id-1", // Has old external_id
	}

	// Kolide still has the old serial
	kolideDevices := []*kolide.Device{
		{
			ID:       "kolide-id-1",
			Serial:   "old-serial", // Different serial
			Platform: "darwin",
		},
	}

	db := database.NewMockDatabase(t)
	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()
	db.EXPECT().UpdateKolideIssues(mock.Anything, mock.Anything).Return(nil).Once()

	var capturedDevices []*pb.Device
	db.EXPECT().UpdateDevices(mock.Anything, mock.MatchedBy(func(devices []*pb.Device) bool {
		capturedDevices = devices
		return true
	})).Return(nil).Once()

	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()

	kolideClient := &mockKolideClient{
		devices: kolideDevices,
		issues:  []*kolide.Issue{},
	}

	sessionStore := auth.NewMockSessionStore(t)
	sessionStore.EXPECT().RefreshDevice(mock.Anything).Return().Maybe()

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, nil, nil, sessionStore, kolideClient, true)

	err := server.UpdateAllDevices(ctx)
	assert.NoError(t, err)

	// The external_id should be cleared since the serial doesn't match anymore
	// THIS ASSERTION WILL FAIL with the current implementation
	assert.Empty(t, capturedDevices[0].ExternalID, "external_id should be cleared when serial no longer matches")
}

// TestUpdateAllDevices_ClearsExternalIDWhenPlatformChanges tests that when a device's
// platform changes, the external_id should be cleared
func TestUpdateAllDevices_ClearsExternalIDWhenPlatformChanges(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup: Device platform changed from linux to darwin
	device := &pb.Device{
		Id:         1,
		Serial:     "serial1",
		Platform:   "darwin", // Platform changed
		Username:   "user@example.com",
		PublicKey:  "publicKey",
		ExternalID: "kolide-id-1", // Has old external_id from when it was linux
	}

	// Kolide has the device with platform linux
	kolideDevices := []*kolide.Device{
		{
			ID:       "kolide-id-1",
			Serial:   "serial1",
			Platform: "linux", // Different platform
		},
	}

	db := database.NewMockDatabase(t)
	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()
	db.EXPECT().UpdateKolideIssues(mock.Anything, mock.Anything).Return(nil).Once()

	var capturedDevices []*pb.Device
	db.EXPECT().UpdateDevices(mock.Anything, mock.MatchedBy(func(devices []*pb.Device) bool {
		capturedDevices = devices
		return true
	})).Return(nil).Once()

	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()

	kolideClient := &mockKolideClient{
		devices: kolideDevices,
		issues:  []*kolide.Issue{},
	}

	sessionStore := auth.NewMockSessionStore(t)
	sessionStore.EXPECT().RefreshDevice(mock.Anything).Return().Maybe()

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, nil, nil, sessionStore, kolideClient, true)

	err := server.UpdateAllDevices(ctx)
	assert.NoError(t, err)

	// The external_id should be cleared since the platform doesn't match anymore
	// THIS ASSERTION WILL FAIL with the current implementation
	assert.Empty(t, capturedDevices[0].ExternalID, "external_id should be cleared when platform no longer matches")
}

// TestUpdateAllDevices_RetainsExternalIDWhenDeviceStillInKolide tests that
// when a device exists in both the database and Kolide with matching serial/platform,
// its external_id should be retained
func TestUpdateAllDevices_RetainsExternalIDWhenDeviceStillInKolide(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup: Device exists with external_id and is still in Kolide
	device := &pb.Device{
		Id:         1,
		Serial:     "serial1",
		Platform:   "darwin",
		Username:   "user@example.com",
		PublicKey:  "publicKey",
		ExternalID: "kolide-id-1",
	}

	// Kolide returns the same device
	kolideDevices := []*kolide.Device{
		{
			ID:       "kolide-id-1",
			Serial:   "serial1",
			Platform: "darwin",
		},
	}

	db := database.NewMockDatabase(t)
	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()
	db.EXPECT().UpdateKolideIssues(mock.Anything, mock.Anything).Return(nil).Once()

	var capturedDevices []*pb.Device
	db.EXPECT().UpdateDevices(mock.Anything, mock.MatchedBy(func(devices []*pb.Device) bool {
		capturedDevices = devices
		return true
	})).Return(nil).Once()

	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()

	kolideClient := &mockKolideClient{
		devices: kolideDevices,
		issues:  []*kolide.Issue{},
	}

	sessionStore := auth.NewMockSessionStore(t)
	sessionStore.EXPECT().RefreshDevice(mock.Anything).Return().Maybe()

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, nil, nil, sessionStore, kolideClient, true)

	err := server.UpdateAllDevices(ctx)
	assert.NoError(t, err)

	// The external_id should be retained since the device is still in Kolide
	assert.Equal(t, "kolide-id-1", capturedDevices[0].ExternalID, "external_id should be retained when device is in Kolide")
}

// TestUpdateAllDevices_SetsExternalIDForNewMatch tests that when a device
// without an external_id matches a Kolide device, the external_id should be set
func TestUpdateAllDevices_SetsExternalIDForNewMatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup: Device without external_id
	device := &pb.Device{
		Id:         1,
		Serial:     "serial1",
		Platform:   "darwin",
		Username:   "user@example.com",
		PublicKey:  "publicKey",
		ExternalID: "", // No external_id yet
	}

	// Kolide has a matching device
	kolideDevices := []*kolide.Device{
		{
			ID:       "kolide-id-1",
			Serial:   "serial1",
			Platform: "darwin",
		},
	}

	db := database.NewMockDatabase(t)
	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()
	db.EXPECT().UpdateKolideIssues(mock.Anything, mock.Anything).Return(nil).Once()

	var capturedDevices []*pb.Device
	db.EXPECT().UpdateDevices(mock.Anything, mock.MatchedBy(func(devices []*pb.Device) bool {
		capturedDevices = devices
		return true
	})).Return(nil).Once()

	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()

	kolideClient := &mockKolideClient{
		devices: kolideDevices,
		issues:  []*kolide.Issue{},
	}

	sessionStore := auth.NewMockSessionStore(t)
	sessionStore.EXPECT().RefreshDevice(mock.Anything).Return().Maybe()

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, nil, nil, sessionStore, kolideClient, true)

	err := server.UpdateAllDevices(ctx)
	assert.NoError(t, err)

	// The external_id should be set since the device now matches
	assert.Equal(t, "kolide-id-1", capturedDevices[0].ExternalID, "external_id should be set for new match")
}

// TestUpdateAllDevices_UpdatesExternalIDWhenMatchChanges tests that when a device
// had an old external_id but now matches a different Kolide device, the external_id
// should be updated to the new match
func TestUpdateAllDevices_UpdatesExternalIDWhenMatchChanges(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup: Device with old external_id
	device := &pb.Device{
		Id:         1,
		Serial:     "serial1",
		Platform:   "darwin",
		Username:   "user@example.com",
		PublicKey:  "publicKey",
		ExternalID: "kolide-id-old", // Old external_id
	}

	// Kolide has a matching device with a new ID (device was recreated in Kolide)
	kolideDevices := []*kolide.Device{
		{
			ID:       "kolide-id-new", // Different ID
			Serial:   "serial1",       // Same serial
			Platform: "darwin",        // Same platform
		},
	}

	db := database.NewMockDatabase(t)
	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()
	db.EXPECT().UpdateKolideIssues(mock.Anything, mock.Anything).Return(nil).Once()

	var capturedDevices []*pb.Device
	db.EXPECT().UpdateDevices(mock.Anything, mock.MatchedBy(func(devices []*pb.Device) bool {
		capturedDevices = devices
		return true
	})).Return(nil).Once()

	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()

	kolideClient := &mockKolideClient{
		devices: kolideDevices,
		issues:  []*kolide.Issue{},
	}

	sessionStore := auth.NewMockSessionStore(t)
	sessionStore.EXPECT().RefreshDevice(mock.Anything).Return().Maybe()

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, nil, nil, sessionStore, kolideClient, true)

	err := server.UpdateAllDevices(ctx)
	assert.NoError(t, err)

	// The external_id should be updated to the new ID
	assert.Equal(t, "kolide-id-new", capturedDevices[0].ExternalID, "external_id should be updated to new matching ID")
}

// TestUpdateAllDevices_HandlesEmptyExternalIDCorrectly tests that devices
// without external_ids that don't match any Kolide device remain without external_ids
func TestUpdateAllDevices_HandlesEmptyExternalIDCorrectly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup: Device without external_id
	device := &pb.Device{
		Id:         1,
		Serial:     "serial1",
		Platform:   "darwin",
		Username:   "user@example.com",
		PublicKey:  "publicKey",
		ExternalID: "", // No external_id
	}

	// Kolide has no matching devices
	kolideDevices := []*kolide.Device{
		{
			ID:       "kolide-id-1",
			Serial:   "different-serial",
			Platform: "linux",
		},
	}

	db := database.NewMockDatabase(t)
	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()
	db.EXPECT().UpdateKolideIssues(mock.Anything, mock.Anything).Return(nil).Once()

	var capturedDevices []*pb.Device
	db.EXPECT().UpdateDevices(mock.Anything, mock.MatchedBy(func(devices []*pb.Device) bool {
		capturedDevices = devices
		return true
	})).Return(nil).Once()

	db.EXPECT().ReadDevices(mock.Anything).Return([]*pb.Device{device}, nil).Once()

	kolideClient := &mockKolideClient{
		devices: kolideDevices,
		issues:  []*kolide.Issue{},
	}

	sessionStore := auth.NewMockSessionStore(t)
	sessionStore.EXPECT().RefreshDevice(mock.Anything).Return().Maybe()

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, nil, nil, sessionStore, kolideClient, true)

	err := server.UpdateAllDevices(ctx)
	assert.NoError(t, err)

	// The external_id should remain empty since there's no match
	assert.Empty(t, capturedDevices[0].ExternalID, "external_id should remain empty when no match found")
}

// mockKolideClient implements the kolide.Client interface for testing
type mockKolideClient struct {
	devices []*kolide.Device
	issues  []*kolide.Issue
	checks  []*kolide.Check
}

func (m *mockKolideClient) GetDevices(ctx context.Context) ([]*kolide.Device, error) {
	return m.devices, nil
}

func (m *mockKolideClient) GetIssues(ctx context.Context) ([]*kolide.Issue, error) {
	return m.issues, nil
}

func (m *mockKolideClient) GetChecks(ctx context.Context) ([]*kolide.Check, error) {
	return m.checks, nil
}

func (m *mockKolideClient) GetDeviceIssues(ctx context.Context, deviceID string) ([]*kolide.Issue, error) {
	return m.issues, nil
}
