package bootstrapper_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nais/device/device-agent/bootstrapper"
	"github.com/stretchr/testify/assert"
)

func TestEnsureBootstrapConfig(t *testing.T) {
	serial, platform := "serial", "platform"
	tunnelIP, publicKey, endpoint, apiserverIP := "tunnelIP", "publicKey", "endpoint", "apiserverIP"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.RequestURI == "/api/v1/deviceinfo" && r.Method == http.MethodPost:
			var di bootstrapper.DeviceInfo
			if err := json.NewDecoder(r.Body).Decode(&di); err != nil {
				assert.NoError(t, err)
			}
			defer r.Body.Close()

			assert.Equal(t, serial, di.Serial)
			assert.Equal(t, platform, di.Platform)

			w.WriteHeader(http.StatusCreated)
		case r.RequestURI == "/api/v1/bootstrapconfig" && r.Method == http.MethodGet:
			bc := bootstrapper.BootstrapConfig{
				DeviceIP:    tunnelIP,
				PublicKey:   publicKey,
				Endpoint:    endpoint,
				APIServerIP: apiserverIP,
			}
			b, err := json.Marshal(&bc)
			assert.NoError(t, err)

			w.Write(b)
		default:
			t.Fatalf("unexpected method on URI: %v %v", r.Method, r.RequestURI)
		}
	}))

	b := bootstrapper.New([]byte("publicKey"), "/some/path", serial, platform, server.URL, server.Client())

	bootstrapConfig, err := b.BootstrapDevice()
	assert.NoError(t, err)
	assert.Equal(t, tunnelIP, bootstrapConfig.DeviceIP)
	assert.Equal(t, publicKey, bootstrapConfig.PublicKey)
	assert.Equal(t, endpoint, bootstrapConfig.Endpoint)
	assert.Equal(t, apiserverIP, bootstrapConfig.APIServerIP)
	fmt.Println(bootstrapConfig)
}
