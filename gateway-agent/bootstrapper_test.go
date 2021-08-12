package gateway_agent_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nais/device/device-agent/filesystem"
	"github.com/nais/device/device-agent/wireguard"
	g "github.com/nais/device/gateway-agent"
	"github.com/nais/device/pkg/bootstrap"
	"github.com/stretchr/testify/assert"
)

func TestGetBootstrapConfig(t *testing.T) {
	t.Run("returns existing bootstrapconfig if present", func(t *testing.T) {
		f, err := ioutil.TempFile(os.TempDir(), "test")
		assert.NoError(t, err)
		defer os.Remove(f.Name())
		defer f.Close()

		deviceIP := "10.255.240.31"
		f.WriteString(`{"deviceIP":"` + deviceIP + `"}`)
		cfg := &g.Config{BootstrapConfigPath: f.Name()}
		bootstrapper := g.Bootstrapper{Config: cfg}
		config, err := bootstrapper.EnsureBootstrapConfig()
		assert.Equal(t, deviceIP, config.DeviceIP)
	})

	t.Run("ensures new bootstrapconfig if not present", func(t *testing.T) {
		privateKey := wireguard.WgGenKey()
		publicKey := wireguard.PublicKey(privateKey)

		gatewayInfo := bootstrap.GatewayInfo{
			Name:      "gateway-test",
			PublicIP:  "13.37.13.37",
			PublicKey: string(publicKey),
		}

		expectedGatewayConfig := bootstrap.Config{
			DeviceIP:       "10.255.240.69",
			PublicKey:      "public-key",
			TunnelEndpoint: "35.35.35.35:51820",
			APIServerIP:    "10.255.240.1",
		}

		handler := http.NewServeMux()
		handler.HandleFunc("/api/v2/gateway/info", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("invalid method for this path")
			}

			var receivedGatewayInfo bootstrap.GatewayInfo
			err := json.NewDecoder(r.Body).Decode(&receivedGatewayInfo)
			assert.NoError(t, err)

			assert.Equal(t, gatewayInfo.Name, receivedGatewayInfo.Name)
			assert.Equal(t, gatewayInfo.PublicIP, receivedGatewayInfo.PublicIP)
			assert.Equal(t, gatewayInfo.PublicKey, receivedGatewayInfo.PublicKey)

			w.WriteHeader(http.StatusCreated)
		})

		specificGatewayConfigUrl := fmt.Sprintf("/api/v2/gateway/config/%s", gatewayInfo.Name)
		handler.HandleFunc(specificGatewayConfigUrl, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("invalid method for this path")
			}

			err := json.NewEncoder(w).Encode(&expectedGatewayConfig)
			assert.NoError(t, err)
		})

		server := httptest.NewServer(handler)

		f, err := ioutil.TempDir(os.TempDir(), "test")
		assert.NoError(t, err)
		defer os.RemoveAll(f)
		bootstrapConfigPath := fmt.Sprintf("%s/bootstrapconfig.json", f)

		cfg := &g.Config{
			PublicIP:            "13.37.13.37",
			BootstrapApiURL:     server.URL,
			Name:                gatewayInfo.Name,
			PrivateKey:          string(privateKey),
			BootstrapConfigPath: bootstrapConfigPath,
		}

		assert.Error(t, filesystem.FileMustExist(bootstrapConfigPath))
		bootstrapper := g.Bootstrapper{Config: cfg, HTTPClient: server.Client()}
		config, err := bootstrapper.EnsureBootstrapConfig()
		assert.NoError(t, err)

		assert.Equal(t, expectedGatewayConfig, *config)

		assert.NoError(t, filesystem.FileMustExist(bootstrapConfigPath))
	})
}
