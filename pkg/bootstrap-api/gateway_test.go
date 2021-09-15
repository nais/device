package bootstrap_api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nais/device/pkg/bootstrap"
	bootstrap_api "github.com/nais/device/pkg/bootstrap-api"
	"github.com/nais/device/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
)

var gatewayInfoUrl string
var gatewayConfigUrl string

const (
	apiserverPassword = "pass"
	apiserverUsername = "user"
)

func TestGatewayEnrollHappyPath(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		t.Fatal()
	}

	token := "fakeToken123"
	gatewayName := "test-gateway"

	sm := &FakeSecretManager{secrets: []*secretmanager.Secret{{
		Name: "/foo/x/y/z" + gatewayName,
		Data: []byte(token),
	}}}

	stop := make(chan struct{}, 1)
	wg := &sync.WaitGroup{}
	server, err := setup(listener, sm, stop, wg)
	assert.NoError(t, err)
	if err != nil {
		t.Fatal()
	}

	gatewayConfigUrl = fmt.Sprintf("http://%s%s", listener.Addr().String(), "/api/v2/gateway/config")
	gatewayInfoUrl = fmt.Sprintf("http://%s%s", listener.Addr().String(), "/api/v2/gateway/info")

	gwInfoToPost := &bootstrap.GatewayInfo{
		Name:     gatewayName,
		PublicIP: "1.2.3.4",
	}

	var gwInfoPostResponse *http.Response

	attempts := 0
	for {
		attempts += 1
		time.Sleep(500 * time.Millisecond)
		fmt.Printf("Attempt %d at posting gateway info", attempts)
		gwInfoPostResponse, err = postGatewayInfo(gatewayName, token, gwInfoToPost)
		if err == nil && gwInfoPostResponse.StatusCode != 401 {
			break
		}

		if attempts >= 10 {
			t.Fatal("reached max attempts for posting gateway info")
		}
	}

	assert.NoError(t, err)
	if err != nil {
		t.Fatal()
	}
	assert.Equal(t, http.StatusCreated, gwInfoPostResponse.StatusCode)

	gwInfos, gwInfosResponse, err := getGatewayInfo()
	assert.NoError(t, err)
	if err != nil {
		t.Fatal()
	}
	assert.Len(t, gwInfos, 1)
	assert.Equal(t, gwInfosResponse.StatusCode, http.StatusOK)

	assert.Equal(t, gwInfos[0].PublicIP, gwInfoToPost.PublicIP)
	assert.Equal(t, gwInfos[0].Name, gwInfoToPost.Name)

	gwConfigToPost := &bootstrap.Config{
		DeviceIP:       "10.255.240.2",
		PublicKey:      "apiserver-public-key",
		TunnelEndpoint: "33.44.55.66:15555",
		APIServerIP:    "10.255.240.1",
	}

	postGwConfigResponse, err := postGatewayConfig(gwConfigToPost, gwInfoToPost.Name)
	assert.NoError(t, err)
	if err != nil {
		t.Fatal()
	}
	assert.Equal(t, http.StatusCreated, postGwConfigResponse.StatusCode)

	returnedGwConfig, getGwConfigResponse, err := getGatewayConfig(gatewayName, token)
	assert.NoError(t, err)
	if err != nil {
		t.Fatal()
	}
	assert.Equal(t, http.StatusOK, getGwConfigResponse.StatusCode)

	assert.Equal(t, returnedGwConfig.APIServerIP, gwConfigToPost.APIServerIP)
	assert.Equal(t, returnedGwConfig.PublicKey, gwConfigToPost.PublicKey)
	assert.Equal(t, returnedGwConfig.DeviceIP, gwConfigToPost.DeviceIP)

	err = server.Close()
	assert.NoError(t, err)
	if err != nil {
		t.Fatal()
	}

	assert.Len(t, sm.secrets, 0, "secret should be disabled")

	stop <- struct{}{}
	wg.Wait()
}

// 1
func postGatewayInfo(gatewayName, token string, config *bootstrap.GatewayInfo) (*http.Response, error) {
	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(*config)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", gatewayInfoUrl, buffer)
	if err != nil {
		return nil, err
	}

	request.SetBasicAuth(gatewayName, token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return response, err
}

// 2
func getGatewayInfo() ([]bootstrap.GatewayInfo, *http.Response, error) {
	request, err := http.NewRequest("GET", gatewayInfoUrl, nil)
	if err != nil {
		return nil, nil, err
	}
	request.SetBasicAuth(apiserverUsername, apiserverPassword)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	var gwInfos []bootstrap.GatewayInfo
	err = json.NewDecoder(response.Body).Decode(&gwInfos)
	if err != nil {
		return nil, nil, err
	}

	return gwInfos, response, err
}

// 3
func postGatewayConfig(config *bootstrap.Config, gatewayName string) (*http.Response, error) {
	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(config)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", gatewayConfigUrl, gatewayName), buffer)
	if err != nil {
		return nil, err
	}

	request.SetBasicAuth(apiserverUsername, apiserverPassword)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return response, err
}

// 4
func getGatewayConfig(gatewayName, token string) (*bootstrap.Config, *http.Response, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", gatewayConfigUrl, gatewayName), nil)
	if err != nil {
		return nil, nil, err
	}
	request.SetBasicAuth(gatewayName, token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	var gwConfig bootstrap.Config
	err = json.NewDecoder(response.Body).Decode(&gwConfig)
	if err != nil {
		return nil, nil, err
	}

	return &gwConfig, response, err
}

func setup(listener net.Listener, sm bootstrap_api.SecretManager, stop chan struct{}, wg *sync.WaitGroup) (*http.Server, error) {
	c := map[string]string{"user": "pass"}
	azureAuthMock := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("mock authed")
			next.ServeHTTP(w, r)
		})
	}

	api := bootstrap_api.NewApi(c, azureAuthMock, sm)

	go func() {
		wg.Add(1)
		api.SyncEnrollmentSecretsLoop(1*time.Second, stop)
		wg.Done()
	}()

	server := &http.Server{Handler: api.Router()}
	go server.Serve(listener)
	time.Sleep(1 * time.Second)

	response, err := http.DefaultClient.Get(fmt.Sprintf("http://%s%s", listener.Addr().String(), "/isalive"))
	if err != nil {
		return nil, nil
	}
	defer response.Body.Close()
	return server, err
}

type FakeSecretManager struct {
	secrets []*secretmanager.Secret
}

func (sm *FakeSecretManager) DisableSecret(name string) error {
	for i, secret := range sm.secrets {
		if !strings.HasSuffix(secret.Name, name) {
			sm.secrets = append(sm.secrets[:i], sm.secrets[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("secret not found: %v", name)
}

func (sm *FakeSecretManager) GetSecrets(filter map[string]string) ([]*secretmanager.Secret, error) {
	return sm.secrets, nil
}
