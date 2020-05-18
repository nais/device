package device_agent

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nais/device/device-agent/config"
	"github.com/stretchr/testify/assert"
)

func TestEnsureBootstrapConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		fmt.Println(err)

		fmt.Println(string(body))
	}))

	deviceAgent := DeviceAgent{
		Client: server.Client(),
		Config: config.Config{
			BootstrapAPI: server.URL,
		},
	}

	bootstrapConfig, err := deviceAgent.EnsureBootstrapConfig()
	//assert.NoError(t, err)
	fmt.Println(err)
	fmt.Println(bootstrapConfig)
}
