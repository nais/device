//go:build integration_test
// +build integration_test

package enroller_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/enroller"
	"github.com/nais/device/pkg/apiserver/testdatabase"
	"github.com/nais/device/pkg/bootstrap"
)

const (
	apiServerPublicKey      = "pk"
	apiserverWireGuardIP    = "10.255.240.1"
	wireguardNetworkAddress = "10.255.240.1/21"
	endpoint                = "ep"
	deviceSerial            = "deviceSerial"
	devicePublicKey         = "publicKey"
	devicePlatform          = "linux"
	deviceOwner             = "me"
	timeout                 = 5 * time.Second
)

func TestWatchDeviceEnrollments(t *testing.T) {
	once := sync.Once{}
	success := false
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v2/device/config/"+deviceSerial, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("invalid method")
		}

		var cfg bootstrap.Config
		err := json.NewDecoder(r.Body).Decode(&cfg)
		assert.NoError(t, err)

		assert.Equal(t, apiServerPublicKey, cfg.PublicKey)
		assert.Equal(t, endpoint, cfg.TunnelEndpoint)
		assert.Equal(t, apiserverWireGuardIP, cfg.APIServerIP)

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
		})

		fmt.Fprint(w, `[]`)
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ipAllocator := database.NewIPAllocator(netip.MustParsePrefix(wireguardNetworkAddress), []string{apiserverWireGuardIP})
	testDB, err := testdatabase.New(ctx, "user=postgres password=postgres host=localhost port=5433 sslmode=disable", ipAllocator)
	assert.NoError(t, err)
	server := httptest.NewServer(mux)
	enr := enroller.Enroller{
		Client:             server.Client(),
		DB:                 testDB,
		BootstrapAPIURL:    server.URL,
		APIServerPublicKey: apiServerPublicKey,
		APIServerEndpoint:  endpoint,
		APIServerIP:        apiserverWireGuardIP,
	}

	assert.NoError(t, enr.EnrollDevice(context.Background()))

	device, err := testDB.ReadDevice(ctx, devicePublicKey)
	assert.NoError(t, err)

	assert.Equal(t, devicePlatform, device.Platform)
	assert.Equal(t, deviceOwner, device.Username)
	assert.Equal(t, devicePublicKey, device.PublicKey)

	if !success {
		t.Errorf("no success")
	}
}
