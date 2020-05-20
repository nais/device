package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi"
	"github.com/nais/device/apiserver/api"
	"github.com/nais/device/apiserver/database"
	"github.com/stretchr/testify/assert"
)

func TestGetDevices(t *testing.T) {
	db, router := setup(t)

	ctx := context.Background()

	publicKey, username, serial, platform := "pubkey", "user", "serial", "darwin"
	if err := db.AddDevice(ctx, username, publicKey, serial, platform); err != nil {
		t.Fatalf("Adding device: %v", err)
	}

	devices := getDevices(t, router)
	assert.Len(t, devices, 1)
	device := devices[0]
	assert.Equal(t, username, device.Username)
	assert.Equal(t, publicKey, device.PublicKey)
	assert.Equal(t, serial, device.Serial)
	assert.Equal(t, platform, device.Platform)
	assert.NotNil(t, device.IP)
	assert.False(t, *device.Healthy, "unhealthy by default")
	assert.Nil(t, device.LastCheck, "unchecked by default")
	assert.Nil(t, device.LastSeen, "unseen by default")
}

func TestUpdateDeviceHealth(t *testing.T) {
	db, router := setup(t)
	device := database.Device{Username: "user@acme.org", Serial: "serial", PublicKey: "pubkey", Platform: "darwin", Healthy: boolp(true)}
	ctx := context.Background()
	if err := db.AddDevice(ctx, device.Username, device.PublicKey, device.Serial, device.Platform); err != nil {
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

func setup(t *testing.T) (*database.APIServerDB, chi.Router) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test")
	}

	db, err := database.NewTestDatabase("postgresql://postgres:postgres@localhost:5433", "../database/schema/schema.sql")
	if err != nil {
		t.Fatalf("Instantiating database: %v", err)
	}
	return db, api.New(api.Config{
		DB:                          db,
		OAuthKeyValidatorMiddleware: TokenValidatorMiddlewareMock([]string{"abc-123", "123-abc"}),
	})
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

func TokenValidatorMiddlewareMock(groups []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), "groups", groups))
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
