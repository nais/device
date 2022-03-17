//go:build integration_test
// +build integration_test

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/auth"
	"github.com/nais/device/pkg/random"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"

	"github.com/nais/device/pkg/apiserver/api"
	apiauth "github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/jita"
	"github.com/nais/device/pkg/apiserver/testdatabase"
	"github.com/nais/device/pkg/pb"
)

const (
	wireguardNetworkAddress = "10.255.240.1/21"
	apiserverWireGuardIP    = "10.255.240.1"
)

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

	authorizedGateway := pb.Gateway{
		Name:           "gw1",
		Endpoint:       "ep1",
		PublicKey:      "pubkey1",
		AccessGroupIDs: []string{"group1"},
	}
	unauthorizedGateway := pb.Gateway{
		Name:      "gw2",
		Endpoint:  "ep2",
		PublicKey: "pubkey2",
	}
	if err := db.AddGateway(ctx, &authorizedGateway); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}

	if err := db.AddGateway(ctx, &unauthorizedGateway); err != nil {
		t.Fatalf("Adding gateway: %v", err)
	}

	gateways := getDeviceConfig(t, router, si.Key)

	assert.Len(t, gateways, 1)
	assert.Equal(t, gateways[0].PublicKey, authorizedGateway.PublicKey)
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
	ipAllocator := database.NewIPAllocator(netip.MustParsePrefix(wireguardNetworkAddress), []string{apiserverWireGuardIP})
	db, err := testdatabase.New(ctx, testDBDSN, ipAllocator)
	if err != nil {
		t.Fatalf("Instantiating database: %v", err)
	}

	sessions := apiauth.NewSessionStore(db)

	authenticator := apiauth.NewAuthenticator(&auth.Azure{}, db, sessions)

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

func executeRequest(req *http.Request, router chi.Router) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}
