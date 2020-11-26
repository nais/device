package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/device/apiserver/jita"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/apiserver/testdatabase"

	"github.com/nais/device/apiserver/auth"

	"github.com/go-chi/chi"
	"github.com/nais/device/apiserver/api"
	"github.com/stretchr/testify/assert"
)

func TestGetDevices(t *testing.T) {
	db, router := setup(t, nil)

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
	db, router := setup(t, nil)

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

	authorizedGateway := database.Gateway{Name: "gw1", Endpoint: "ep1", PublicKey: "pubkey1"}
	unauthorizedGateway := database.Gateway{Name: "gw2", Endpoint: "ep2", PublicKey: "pubkey2"}
	if err := db.AddGateway(ctx, authorizedGateway.Name, authorizedGateway.Endpoint, authorizedGateway.PublicKey); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}

	assert.NoError(t, db.UpdateGateway(ctx, authorizedGateway.Name, nil, []string{"group1"}, false))

	if err := db.AddGateway(ctx, unauthorizedGateway.Name, unauthorizedGateway.Endpoint, unauthorizedGateway.PublicKey); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}

	gateways := getDeviceConfig(t, router, "keyyolo123")

	assert.Len(t, gateways, 1)
	assert.Equal(t, gateways[0].PublicKey, authorizedGateway.PublicKey)
}

func TestGatewayConfig(t *testing.T) {
	db, router := setup(t, nil)

	ctx := context.Background()

	healthyDevice := addDevice(t, db, ctx, "serial1", "healthyUser", "pubKey1", true)
	healthyDevice2 := addDevice(t, db, ctx, "serial2", "healthyUser2", "pubKey2", true)
	unhealthyDevice := addDevice(t, db, ctx, "serial3", "unhealthyUser", "pubKey3", false)

	_ = addSessionInfo(t, db, ctx, healthyDevice, "userId", []string{"authorized"})
	_ = addSessionInfo(t, db, ctx, healthyDevice2, "userId", []string{"unauthorized"})
	_ = addSessionInfo(t, db, ctx, unhealthyDevice, "userId", []string{"authorized"})
	_ = addSessionInfo(t, db, ctx, unhealthyDevice, "userId", []string{"unauthorized"})
	_ = addSessionInfo(t, db, ctx, healthyDevice2, "userId", []string{""})

	// todo don't use username as gateway
	authorizedGateway := database.Gateway{Name: "username", Endpoint: "ep1", PublicKey: "pubkey1"}
	if err := db.AddGateway(ctx, authorizedGateway.Name, authorizedGateway.Endpoint, authorizedGateway.PublicKey); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}
	assert.NoError(t, db.UpdateGateway(ctx, authorizedGateway.Name, nil, []string{"authorized"}, false))

	gatewayConfig := getGatewayConfig(t, router, "username", "password")
	devices := gatewayConfig.Devices

	assert.Len(t, devices, 1)
	assert.Equal(t, devices[0].PublicKey, healthyDevice.PublicKey)
}

func TestPrivilegedGatewayConfig(t *testing.T) {
	api.InitializeMetrics("test")
	ctx := context.Background()

	privilegedUsers := []jita.PrivilegedUser{{
		UserId: "userId",
	}}
	server := httptest.NewServer(mockJita(t, "privileged1", privilegedUsers))
	db, router := setup(t, jita.New("username", "password", server.URL))

	healthyDevice := addDevice(t, db, ctx, "serial1", "healthyUser", "pubKey1", true)

	_ = addSessionInfo(t, db, ctx, healthyDevice, privilegedUsers[0].UserId, []string{"authorized"})

	// todo don't use username as gateway
	privilegedGateway1 := database.Gateway{Name: "privileged1", Endpoint: "ep1", PublicKey: "pubkey1"}
	if err := db.AddGateway(ctx, privilegedGateway1.Name, privilegedGateway1.Endpoint, privilegedGateway1.PublicKey); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}
	assert.NoError(t, db.UpdateGateway(ctx, privilegedGateway1.Name, nil, []string{"authorized"}, true))

	privilegedGateway2 := database.Gateway{Name: "privileged2", Endpoint: "ep1", PublicKey: "pubkey2"}
	if err := db.AddGateway(ctx, privilegedGateway2.Name, privilegedGateway2.Endpoint, privilegedGateway2.PublicKey); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}
	assert.NoError(t, db.UpdateGateway(ctx, privilegedGateway2.Name, nil, []string{"authorized"}, true))

	privilegedGatewayConfig := getGatewayConfig(t, router, "privileged1", "password")
	assert.Len(t, privilegedGatewayConfig.Devices, 1)
	assert.Equal(t, privilegedGatewayConfig.Devices[0].PublicKey, healthyDevice.PublicKey)

	unprivilegedGatewayConfig := getGatewayConfig(t, router, "privileged2", "password")
	assert.Len(t, unprivilegedGatewayConfig.Devices, 0)

	server.Close()
}

func addDevice(t *testing.T, db *database.APIServerDB, ctx context.Context, serial, username, publicKey string, healthy bool) *database.Device {
	device := database.Device{
		Serial:    serial,
		PublicKey: publicKey,
		Username:  username,
		Platform:  "darwin",
	}

	if err := db.AddDevice(ctx, device); err != nil {
		t.Fatalf("Adding device: %v", err)
		return nil
	}

	if healthy {
		device.Healthy = boolp(true)

		if err := db.UpdateDeviceStatus([]database.Device{device}); err != nil {
			t.Fatalf("Updating device status: %v", err)
			return nil
		}
	}

	deviceWithId, err := db.ReadDevice(device.PublicKey)
	if err != nil {
		t.Fatalf("Reading device: %v", err)
	}

	return deviceWithId
}

func addSessionInfo(t *testing.T, db *database.APIServerDB, ctx context.Context, databaseDevice *database.Device, userId string, groups []string) *database.SessionInfo {
	databaseSessionInfo := database.SessionInfo{
		Key:      "dbSessionKey",
		Expiry:   time.Now().Add(time.Minute).Unix(),
		Device:   databaseDevice,
		Groups:   groups,
		ObjectId: userId,
	}
	if err := db.AddSessionInfo(ctx, &databaseSessionInfo); err != nil {
		t.Fatalf("Adding SessionInfo: %v", err)
	}

	return &databaseSessionInfo
}

func TestUpdateDeviceHealth(t *testing.T) {
	db, router := setup(t, nil)
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
	db, router := setup(t, nil)

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

func mockJita(t *testing.T, gatewayName string, privilegedUsers []jita.PrivilegedUser) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc(fmt.Sprintf("/api/v1/gatewayAccess/%s", gatewayName), func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("invalid method")
		}

		err := json.NewEncoder(w).Encode(privilegedUsers)
		assert.NoError(t, err)
	})
	return mux
}

func setup(t *testing.T, j *jita.Jita) (*database.APIServerDB, chi.Router) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test")
	}
	ctx := context.Background()

	testDBDSN := "user=postgres password=postgres host=localhost port=5433 sslmode=disable"

	db, err := testdatabase.New(ctx, testDBDSN)
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
		DB:   db,
		Jita: j,
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

func getGatewayConfig(t *testing.T, router chi.Router, username, password string) api.GatewayConfig {
	req, _ := http.NewRequest("GET", "/gatewayconfig", nil)
	req.SetBasicAuth(username, password)
	resp := executeRequest(req, router)
	assert.Equal(t, http.StatusOK, resp.Code)

	var gatewayConfig api.GatewayConfig
	if err := json.NewDecoder(resp.Body).Decode(&gatewayConfig); err != nil {
		t.Fatalf("Unmarshaling response body: %v", err)
	}

	return gatewayConfig
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
