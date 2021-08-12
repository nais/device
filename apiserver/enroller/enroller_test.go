package enroller_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/nais/device/apiserver/enroller"
	"github.com/nais/device/apiserver/testdatabase"
	"github.com/nais/device/pkg/bootstrap"
	"github.com/stretchr/testify/assert"
)

const (
	apiServerPublicKey = "pk"
	endpoint           = "ep"
	gatewayName        = "gateway-1"
	gatewayEndpoint    = "1.2.3.4"
	gatewayPublicKey   = "publicKey"
	deviceSerial       = "deviceSerial"
	devicePublicKey    = "publicKey"
	devicePlatform     = "linux"
	deviceOwner        = "me"
)

func TestWatchGatewayEnrollments(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test")
	}

	once := sync.Once{}
	success := false
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v2/gateway/config/gateway-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("invalid method")
		}

		var cfg bootstrap.Config
		err := json.NewDecoder(r.Body).Decode(&cfg)
		assert.NoError(t, err)

		assert.Equal(t, apiServerPublicKey, cfg.PublicKey)
		assert.Equal(t, endpoint, cfg.TunnelEndpoint)

		success = true
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("/api/v2/gateway/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("invalid method")
		}

		gwInfos := []bootstrap.GatewayInfo{{
			Name:      gatewayName,
			PublicIP:  gatewayEndpoint,
			PublicKey: gatewayPublicKey,
		}}

		b, err := json.Marshal(&gwInfos)
		assert.NoError(t, err)

		once.Do(func() {
			w.Write(b)
			return
		})

		fmt.Fprint(w, `[]`)
	})
	ctx := context.Background()

	testDB, err := testdatabase.New(ctx, "user=postgres password=postgres host=localhost port=5433 sslmode=disable")
	assert.NoError(t, err)
	server := httptest.NewServer(mux)
	enr := enroller.Enroller{
		Client:             server.Client(),
		DB:                 testDB,
		BootstrapAPIURL:    server.URL,
		APIServerPublicKey: apiServerPublicKey,
		APIServerEndpoint:  endpoint,
	}

	assert.NoError(t, enr.EnrollGateways(context.Background()))

	gateway, err := testDB.ReadGateway(gatewayName)
	assert.NoError(t, err)

	assert.Equal(t, gatewayEndpoint, gateway.Endpoint)
	assert.Equal(t, gatewayPublicKey, gateway.PublicKey)

	if !success {
		t.Errorf("no success")
	}
}

func TestWatchDeviceEnrollments(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("Skipping integration test")
	}

	once := sync.Once{}
	success := false
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v2/device/config/deviceSerial", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("invalid method")
		}

		var cfg bootstrap.Config
		err := json.NewDecoder(r.Body).Decode(&cfg)
		assert.NoError(t, err)

		assert.Equal(t, apiServerPublicKey, cfg.PublicKey)
		assert.Equal(t, endpoint, cfg.TunnelEndpoint)

		success = true
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("/api/v2/device/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("invalid method")
		}

		deviceInfos := []bootstrap.DeviceInfo{{
			Serial:    deviceSerial,
			PublicKey: devicePublicKey,
			Platform:  devicePlatform,
			Owner:     deviceOwner,
		}}

		b, err := json.Marshal(&deviceInfos)
		assert.NoError(t, err)

		once.Do(func() {
			w.Write(b)
			return
		})

		fmt.Fprint(w, `[]`)
	})
	ctx := context.Background()
	testDB, err := testdatabase.New(ctx, "user=postgres password=postgres host=localhost port=5433 sslmode=disable")
	assert.NoError(t, err)
	server := httptest.NewServer(mux)
	enr := enroller.Enroller{
		Client:             server.Client(),
		DB:                 testDB,
		BootstrapAPIURL:    server.URL,
		APIServerPublicKey: apiServerPublicKey,
		APIServerEndpoint:  endpoint,
	}

	assert.NoError(t, enr.EnrollDevice(context.Background()))

	device, err := testDB.ReadDevice(devicePublicKey)
	assert.NoError(t, err)

	assert.Equal(t, devicePlatform, device.Platform)
	assert.Equal(t, deviceOwner, device.Username)
	assert.Equal(t, devicePublicKey, device.PublicKey)

	if !success {
		t.Errorf("no success")
	}
}
