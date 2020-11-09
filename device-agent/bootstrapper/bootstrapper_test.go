package bootstrapper_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nais/device/pkg/bootstrap"

	"github.com/nais/device/device-agent/bootstrapper"
	"github.com/stretchr/testify/assert"
)

func TestBootstrapDevice(t *testing.T) {
	serial, platform := "serial", "platform"
	tunnelIP, publicKey, endpoint, apiserverIP := "tunnelIP", "publicKey", "endpoint", "apiserverIP"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.RequestURI == "/api/v2/device/info" && r.Method == http.MethodPost:
			var di bootstrap.DeviceInfo
			if err := json.NewDecoder(r.Body).Decode(&di); err != nil {
				assert.NoError(t, err)
			}
			defer r.Body.Close()

			assert.Equal(t, serial, di.Serial)
			assert.Equal(t, platform, di.Platform)

			w.WriteHeader(http.StatusCreated)
		case strings.HasPrefix(r.RequestURI, "/api/v2/device/config/") && r.Method == http.MethodGet:
			bc := bootstrap.Config{
				DeviceIP:       tunnelIP,
				PublicKey:      publicKey,
				TunnelEndpoint: endpoint,
				APIServerIP:    apiserverIP,
			}
			b, err := json.Marshal(&bc)
			assert.NoError(t, err)

			w.Write(b)
		default:
			t.Fatalf("unexpected method on URI: %v %v", r.Method, r.RequestURI)
		}
	}))

	di := &bootstrap.DeviceInfo{
		Serial:    serial,
		PublicKey: publicKey,
		Platform:  platform,
	}

	bootstrapConfig, err := bootstrapper.BootstrapDevice(di, server.URL, server.Client())

	assert.NoError(t, err)
	assert.Equal(t, tunnelIP, bootstrapConfig.DeviceIP)
	assert.Equal(t, publicKey, bootstrapConfig.PublicKey)
	assert.Equal(t, endpoint, bootstrapConfig.TunnelEndpoint)
	assert.Equal(t, apiserverIP, bootstrapConfig.APIServerIP)
}
