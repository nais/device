package enroller_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/device/apiserver/enroller"
	"github.com/nais/device/apiserver/testdatabase"
	"github.com/nais/device/pkg/bootstrap"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

const (
	PublicKey = "pk"
	Endpoint  = "ep"
	Timeout   = 3 * time.Second
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

		assert.Equal(t, PublicKey, cfg.PublicKey)
		assert.Equal(t, Endpoint, cfg.TunnelEndpoint)

		success = true
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("/api/v2/gateway/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("invalid method")
		}

		once.Do(func() {
			fmt.Fprint(w, `[{"name": "gateway-1", "publicKey": "pubkey", "endpoint": "1.2.3.4"}]`)
			return
		})

		fmt.Fprint(w, `[]`)
	})

	testDB, err := testdatabase.New("user=postgres password=postgres host=localhost port=5433 sslmode=disable", "../database/schema/schema.sql")
	assert.NoError(t, err)
	server := httptest.NewServer(mux)
	enroller := enroller.Enroller{
		Client:             server.Client(),
		DB:                 testDB,
		BootstrapAPIURL:    server.URL,
		APIServerPublicKey: PublicKey,
		APIServerEndpoint:  Endpoint,
	}

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	enroller.WatchGatewayEnrollments(ctx)

	device, err := testDB.ReadGateway("gateway-1")
	assert.NoError(t, err)

	assert.Equal(t, "1.2.3.4", device.Endpoint)
	assert.Equal(t, "pubkey", device.PublicKey)

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

	mux.HandleFunc("/api/v2/device/config/serial", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("invalid method")
		}

		var cfg bootstrap.Config
		err := json.NewDecoder(r.Body).Decode(&cfg)
		assert.NoError(t, err)

		assert.Equal(t, PublicKey, cfg.PublicKey)
		assert.Equal(t, Endpoint, cfg.TunnelEndpoint)

		success = true
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("/api/v2/device/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("invalid method")
		}

		once.Do(func() {
			fmt.Fprint(w, `[{"serial": "serial", "publicKey": "pubkey", "platform": "linux", "owner": "me"}]`)
			return
		})

		fmt.Fprint(w, `[]`)
	})

	testDB, err := testdatabase.New("user=postgres password=postgres host=localhost port=5433 sslmode=disable", "../database/schema/schema.sql")
	assert.NoError(t, err)
	server := httptest.NewServer(mux)
	enroller := enroller.Enroller{
		Client:             server.Client(),
		DB:                 testDB,
		BootstrapAPIURL:    server.URL,
		APIServerPublicKey: PublicKey,
		APIServerEndpoint:  Endpoint,
	}

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	enroller.WatchDeviceEnrollments(ctx)

	device, err := testDB.ReadDevice("pubkey")
	assert.NoError(t, err)

	assert.Equal(t, "linux", device.Platform)
	assert.Equal(t, "me", device.Username)

	if !success {
		t.Errorf("no success")
	}
}
