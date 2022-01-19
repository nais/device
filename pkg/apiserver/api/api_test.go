//go:build integration_test
// +build integration_test

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/azure"

	"github.com/nais/device/pkg/random"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/nais/device/pkg/apiserver/api"
	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/jita"
	"github.com/nais/device/pkg/apiserver/testdatabase"
	"github.com/nais/device/pkg/pb"
)

func TestGetDevices(t *testing.T) {
	db, router := setup(t, nil)

	ctx := context.Background()

	d := &pb.Device{Username: "user", Serial: "serial", PublicKey: "pubkey", Platform: "darwin"}
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
	assert.NotNil(t, device.Ip)
	assert.False(t, device.Healthy, "unhealthy by default")
	assert.Nil(t, device.LastUpdated, "not updated by default")
	assert.Nil(t, device.KolideLastSeen, "unseen by default")
}

func TestGetDeviceConfig(t *testing.T) {
	db, router := setup(t, nil)

	ctx := context.Background()

	device := &pb.Device{
		Serial:    "serial",
		PublicKey: "pubkey",
		Username:  "user",
		Platform:  "darwin",
	}

	if err := db.AddDevice(ctx, device); err != nil {
		t.Fatalf("Adding device: %v", err)
	}

	device.Healthy = true

	err := db.UpdateDevices(ctx, []*pb.Device{device})
	assert.NoError(t, err)
	si := &pb.Session{
		Key:    "key",
		Device: device,
		Expiry: timestamppb.New(time.Now().Add(time.Minute)),
		Groups: []string{"group1"},
	}

	err = db.AddSessionInfo(ctx, si)
	assert.NoError(t, err)

	authorizedGateway := pb.Gateway{Name: "gw1", Endpoint: "ep1", PublicKey: "pubkey1"}
	unauthorizedGateway := pb.Gateway{Name: "gw2", Endpoint: "ep2", PublicKey: "pubkey2"}
	if err := db.AddGateway(ctx, authorizedGateway.Name, authorizedGateway.Endpoint, authorizedGateway.PublicKey); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}

	assert.NoError(t, db.UpdateGateway(ctx, authorizedGateway.Name, nil, []string{"group1"}, false))

	if err := db.AddGateway(ctx, unauthorizedGateway.Name, unauthorizedGateway.Endpoint, unauthorizedGateway.PublicKey); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}

	gateways := getDeviceConfig(t, router, si.Key)

	assert.Len(t, gateways, 1)
	assert.Equal(t, gateways[0].PublicKey, authorizedGateway.PublicKey)
}

func TestGatewayConfig(t *testing.T) {
	db, router := setup(t, nil)

	ctx := context.Background()

	healthyDevice := addDevice(t, db, ctx, "serial1", "healthyUser", "pubKey1", true, time.Now())
	healthyDevice2 := addDevice(t, db, ctx, "serial2", "healthyUser2", "pubKey2", true, time.Now())
	// healthyDeviceOutOfDate := addDevice(t, db, ctx, "serial2", "healthyUser2", "pubKey2", true, time.Now().Add(-api.MaxTimeSinceKolideLastSeen))
	unhealthyDevice := addDevice(t, db, ctx, "serial3", "unhealthyUser", "pubKey3", false, time.Now())

	_ = addSessionInfo(t, db, ctx, healthyDevice, "userId", []string{"authorized"})
	_ = addSessionInfo(t, db, ctx, healthyDevice2, "userId", []string{"unauthorized"})
	_ = addSessionInfo(t, db, ctx, unhealthyDevice, "userId", []string{"authorized"})
	_ = addSessionInfo(t, db, ctx, unhealthyDevice, "userId", []string{"unauthorized"})
	_ = addSessionInfo(t, db, ctx, healthyDevice2, "userId", []string{""})
	//_ = addSessionInfo(t, db, ctx, healthyDeviceOutOfDate, "userId", []string{"authorized"})

	// todo don't use username as gateway
	authorizedGateway := pb.Gateway{Name: "username", Endpoint: "ep1", PublicKey: "pubkey1"}
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
	ctx := context.Background()

	privilegedUsers := []jita.PrivilegedUser{{
		UserId: "userId",
	}}
	server := httptest.NewServer(mockJita(t, "privileged1", privilegedUsers))
	db, router := setup(t, jita.New("username", "password", server.URL))

	healthyDevice := addDevice(t, db, ctx, "serial1", "healthyUser", "pubKey1", true, time.Now())

	_ = addSessionInfo(t, db, ctx, healthyDevice, privilegedUsers[0].UserId, []string{"authorized"})

	// todo don't use username as gateway
	privilegedGateway1 := pb.Gateway{Name: "privileged1", Endpoint: "ep1", PublicKey: "pubkey1"}
	if err := db.AddGateway(ctx, privilegedGateway1.Name, privilegedGateway1.Endpoint, privilegedGateway1.PublicKey); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}
	assert.NoError(t, db.UpdateGateway(ctx, privilegedGateway1.Name, nil, []string{"authorized"}, true))

	privilegedGateway2 := pb.Gateway{Name: "privileged2", Endpoint: "ep1", PublicKey: "pubkey2"}
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

func addDevice(t *testing.T, db database.APIServer, ctx context.Context, serial, username, publicKey string, healthy bool, lastSeen time.Time) *pb.Device {
	device := &pb.Device{
		Serial:         serial,
		PublicKey:      publicKey,
		Username:       username,
		Platform:       "darwin",
		KolideLastSeen: timestamppb.New(lastSeen),
	}

	if err := db.AddDevice(ctx, device); err != nil {
		t.Fatalf("Adding device: %v", err)
		return nil
	}

	if healthy {
		device.Healthy = true

		if err := db.UpdateDevices(ctx, []*pb.Device{device}); err != nil {
			t.Fatalf("Updating device status: %v", err)
			return nil
		}
	}

	deviceWithId, err := db.ReadDevice(ctx, device.PublicKey)
	if err != nil {
		t.Fatalf("Reading device: %v", err)
	}

	return deviceWithId
}

func addSessionInfo(t *testing.T, db database.APIServer, ctx context.Context, databaseDevice *pb.Device, userId string, groups []string) *pb.Session {
	databaseSessionInfo := &pb.Session{
		Key:      random.RandomString(20, random.LettersAndNumbers),
		Expiry:   timestamppb.New(time.Now().Add(time.Minute)),
		Device:   databaseDevice,
		Groups:   groups,
		ObjectID: userId,
	}
	if err := db.AddSessionInfo(ctx, databaseSessionInfo); err != nil {
		t.Fatalf("Adding SessionInfo: %v", err)
	}

	return databaseSessionInfo
}

func TestGetDeviceConfigSessionNotInCache(t *testing.T) {
	db, router := setup(t, nil)

	ctx := context.Background()

	device := &pb.Device{
		Serial:    "serial",
		PublicKey: "pubkey",
		Username:  "user",
		Platform:  "darwin",
	}

	if err := db.AddDevice(ctx, device); err != nil {
		t.Fatalf("Adding device: %v", err)
	}

	device.Healthy = true

	err := db.UpdateDevices(ctx, []*pb.Device{device})
	assert.NoError(t, err)

	// Read from db as we need the device ID
	databaseDevice, err := db.ReadDevice(ctx, device.PublicKey)
	if err != nil {
		t.Fatalf("Reading device from db: %v", err)
	}

	databaseSessionInfo := &pb.Session{
		Key:    "dbSessionKey",
		Expiry: timestamppb.New(time.Now().Add(time.Minute)),
		Device: databaseDevice,
		Groups: []string{"group1", "group2"},
	}
	if err := db.AddSessionInfo(ctx, databaseSessionInfo); err != nil {
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

func setup(t *testing.T, j jita.Client) (database.APIServer, chi.Router) {
	ctx := context.Background()

	testDBDSN := "user=postgres password=postgres host=localhost port=5433 sslmode=disable"

	db, err := testdatabase.New(ctx, testDBDSN)
	if err != nil {
		t.Fatalf("Instantiating database: %v", err)
	}

	sessions := auth.NewSessionStore(db)

	authenticator := auth.NewAuthenticator(&azure.Azure{}, db, sessions)

	return db, api.New(api.Config{
		DB:            db,
		Jita:          j,
		Authenticator: authenticator,
	})
}

func getDeviceConfig(t *testing.T, router chi.Router, sessionKey string) (gateways []pb.Gateway) {
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

func getDevices(t *testing.T, router chi.Router) (devices []*pb.Device) {
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
