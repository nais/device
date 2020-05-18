package bootstrapper_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nais/device/device-agent/bootstrapper"
	"github.com/stretchr/testify/assert"
)

func TestEnsureBootstrapConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		fmt.Println(err)

		fmt.Println(string(body))
		// unmarshal into deviceinfo, verify data.

		// return canned response on different URL
		w.WriteHeader(http.StatusCreated)
	}))

	b := bootstrapper.New([]byte("publicKey"), "/some/path", "serial", "platform", server.URL, server.Client())

	bootstrapConfig, err := b.EnsureBootstrapConfig()
	//assert.NoError(t, err)
	fmt.Println(err)
	fmt.Println(bootstrapConfig)
}
