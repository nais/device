package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/apiserver/testdatabase"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/nais/device/apiserver/auth"

	"github.com/go-chi/chi"
	"github.com/nais/device/apiserver/api"
	"github.com/stretchr/testify/assert"
)

func TestGetDevices(t *testing.T) {
	db, router := setup(t)

	ctx := context.Background()

	d := database.Device{Username: "user", Serial: "serial", PublicKey: "pubkey", Platform: "darwin"}
	if err := db.AddDevice(ctx, d); err != nil {
		t.Fatalf("Adding device: %v", err)
	}

	devices := getDevices(t, router)
	assert.Len(t, devices, 1)
	device := devices[0]
	assert.Equal(t, d.Username, device.Username)
	assert.Equal(t, d.PublicKey, device.PublicKey)
	assert.Equal(t, d.Serial, device.Serial)
	assert.Equal(t, d.Platform, device.Platform)
	assert.NotNil(t, device.IP)
	assert.False(t, *device.Healthy, "unhealthy by default")
	assert.Nil(t, device.LastUpdated, "not updated by default")
	assert.Nil(t, device.KolideLastSeen, "unseen by default")
}

func TestGetDeviceConfig(t *testing.T) {
	db, router := setup(t)

	ctx := context.Background()

	device := database.Device{
		Serial:    "serial",
		PublicKey: "pubkey",
		Username:  "user",
		Platform:  "darwin",
	}

	if err := db.AddDevice(ctx, device); err != nil {
		t.Fatalf("Adding device: %v", err)
	}

	device.Healthy = boolp(true)

	err := db.UpdateDeviceStatus([]database.Device{device})
	assert.NoError(t, err)

	authorizedGateway := database.Gateway{Name: "gw1", AccessGroupIDs: []string{"group1"}, PublicKey: "pubkey1", IP: "1.2.3.4"}
	unauthorizedGateway := database.Gateway{Name: "gw2", AccessGroupIDs: []string{"group2"}, PublicKey: "pubkey2", IP: "1.2.3.5"}

	if err := db.AddGateway(ctx, authorizedGateway); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}
	if err := db.AddGateway(ctx, unauthorizedGateway); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}

	gateways := getDeviceConfig(t, router, "keyyolo123")

	assert.Len(t, gateways, 1)
	assert.Equal(t, gateways[0].PublicKey, authorizedGateway.PublicKey)
}

func TestUpdateDeviceHealth(t *testing.T) {
	db, router := setup(t)
	device := database.Device{Username: "user@acme.org", Serial: "serial", PublicKey: "pubkey", Platform: "darwin", Healthy: boolp(true)}
	ctx := context.Background()
	if err := db.AddDevice(ctx, device); err != nil {
		t.Fatalf("Adding device: %v", err)
	}

	devices := getDevices(t, router)
	assert.Len(t, devices, 1)
	assert.False(t, *devices[0].Healthy)

	devicesJSON := []database.Device{device}
	b, err := json.Marshal(&devicesJSON)
	if err != nil {
		t.Fatalf("Marshalling device JSON: %v", err)
	}

	req, _ := http.NewRequest("PUT", "/devices/health", bytes.NewReader(b))
	resp := executeRequest(req, router)
	assert.Equal(t, http.StatusOK, resp.Code)

	devices = getDevices(t, router)

	assert.Len(t, devices, 1)
	assert.True(t, *devices[0].Healthy)
}

func TestGetDeviceConfigSessionNotInCache(t *testing.T) {
	db, router := setup(t)

	ctx := context.Background()

	device := database.Device{
		Serial:    "serial",
		PublicKey: "pubkey",
		Username:  "user",
		Platform:  "darwin",
	}

	if err := db.AddDevice(ctx, device); err != nil {
		t.Fatalf("Adding device: %v", err)
	}

	device.Healthy = boolp(true)

	err := db.UpdateDeviceStatus([]database.Device{device})
	assert.NoError(t, err)

	// Read from db as we need the device ID
	databaseDevice, err := db.ReadDevice(device.PublicKey)
	if err != nil {
		t.Fatalf("Reading device from db: %v", err)
	}

	databaseSessionInfo := database.SessionInfo{
		Key:    "dbSessionKey",
		Expiry: time.Now().Add(time.Minute).Unix(),
		Device: databaseDevice,
		Groups: []string{"group1", "group2"},
	}
	if err := db.AddSessionInfo(ctx, &databaseSessionInfo); err != nil {
		t.Fatalf("Adding SessionInfo: %v", err)
	}

	gateways := getDeviceConfig(t, router, "dbSessionKey")

	assert.Len(t, gateways, 0)
}

func setup(t *testing.T) (*database.APIServerDB, chi.Router) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test")
	}

	testDBDSN := "user=postgres password=postgres host=localhost port=5433 sslmode=disable"

	db, err := testdatabase.New(testDBDSN, "../database/schema/schema.sql")
	if err != nil {
		t.Fatalf("Instantiating database: %v", err)
	}

	sessionInfo := database.SessionInfo{
		Key:    "keyyolo123",
		Expiry: time.Now().Add(1 * time.Minute).Unix(),
		Device: &database.Device{
			ID:       1,
			Serial:   "serial",
			Platform: "platform",
			Username: "username",
		},
		Groups: []string{"group1"},
	}

	assert.NoError(t, err)

	return db, api.New(api.Config{
		DB: db,
		Sessions: &auth.Sessions{
			DB:     db,
			Active: map[string]*database.SessionInfo{sessionInfo.Key: &sessionInfo},
		},
	})
}

func getDeviceConfig(t *testing.T, router chi.Router, sessionKey string) (gateways []database.Gateway) {
	req, _ := http.NewRequest("GET", "/deviceconfig", nil)
	req.Header.Add("x-naisdevice-session-key", sessionKey)
	resp := executeRequest(req, router)
	assert.Equal(t, http.StatusOK, resp.Code)

	if err := json.NewDecoder(resp.Body).Decode(&gateways); err != nil {
		t.Fatalf("Unmarshaling response body: %v", err)
	}

	return gateways
}

func getDevices(t *testing.T, router chi.Router) (devices []database.Device) {
	req, _ := http.NewRequest("GET", "/devices", nil)
	resp := executeRequest(req, router)
	assert.Equal(t, http.StatusOK, resp.Code)

	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		t.Fatalf("Unmarshaling response body: %v", err)
	}

	return devices
}

func executeRequest(req *http.Request, router chi.Router) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

func boolp(b bool) *bool {
	return &b
}
