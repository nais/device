package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	main "github.com/nais/device/cmd/bootstrap-api"
	"github.com/nais/device/pkg/bootstrap"
	"github.com/stretchr/testify/assert"
	"log"
	"net"
	"net/http"
	"testing"
	"time"
)

var gatewayInfoUrl string
var gatewayConfigUrl string
var tokenUrl string

func TestGatewayEnrollHappyPath(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	server, err := setup(listener)
	assert.NoError(t, err)

	tokenUrl = fmt.Sprintf("http://%s%s", listener.Addr().String(), "/api/v1/token")
	gatewayConfigUrl = fmt.Sprintf("http://%s%s", listener.Addr().String(), "/api/v1/gatewayconfig")
	gatewayInfoUrl = fmt.Sprintf("http://%s%s", listener.Addr().String(), "/api/v1/gatewayinfo")

	token := "fakeToken123"
	tokenResponse, err := addToken(token)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, tokenResponse.StatusCode)

	gwInfoToPost := &bootstrap.GatewayInfo{
		Name:     "test",
		PublicIP: "1.2.3.4",
	}

	gwInfoPostResponse, err := postGatewayInfo(token, gwInfoToPost)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, gwInfoPostResponse.StatusCode)

	gwInfos, gwInfosResponse, err := getGatewayInfo()
	assert.NoError(t, err)
	assert.Len(t, gwInfos, 1)
	assert.Equal(t, gwInfosResponse.StatusCode, http.StatusOK)

	assert.Equal(t, gwInfos[0].PublicIP, gwInfoToPost.PublicIP)
	assert.Equal(t, gwInfos[0].Name, gwInfoToPost.Name)

	gwConfigToPost := &bootstrap.GatewayConfig{
		TunnelIP:           "10.255.240.2",
		APIServerPublicKey: "apiserver-public-key",
		APIServerIP:        "33.44.55.66",
	}

	postGwConfigResponse, err := postGatewayConfig(token, gwConfigToPost)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, postGwConfigResponse.StatusCode)

	returnedGwConfig, getGwConfigResponse, err := getGatewayConfig(token)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, getGwConfigResponse.StatusCode)

	assert.Equal(t, returnedGwConfig.APIServerIP, gwConfigToPost.APIServerIP)
	assert.Equal(t, returnedGwConfig.APIServerPublicKey, gwConfigToPost.APIServerPublicKey)
	assert.Equal(t, returnedGwConfig.TunnelIP, gwConfigToPost.TunnelIP)

	err = server.Close()
	assert.NoError(t, err)
	time.Sleep(1 * time.Second)
}

func postGatewayInfo(token string, config *bootstrap.GatewayInfo) (*http.Response, error) {
	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(*config)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", gatewayInfoUrl, buffer)
	if err != nil {
		return nil, err
	}

	request.Header.Add(main.TokenHeaderKey, token)
	request.SetBasicAuth("user", "pass")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return response, err
}

func getGatewayInfo() ([]bootstrap.GatewayInfo, *http.Response, error) {
	request, err := http.NewRequest("GET", gatewayInfoUrl, nil)
	if err != nil {
		return nil, nil, err
	}
	request.SetBasicAuth("user", "pass")

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

func postGatewayConfig(token string, config *bootstrap.GatewayConfig) (*http.Response, error) {
	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(config)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest("POST", gatewayConfigUrl, buffer)
	if err != nil {
		return nil, err
	}

	request.Header.Add(main.TokenHeaderKey, token)
	request.SetBasicAuth("user", "pass")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return response, err
}

func getGatewayConfig(token string) (*bootstrap.GatewayConfig, *http.Response, error) {
	request, err := http.NewRequest("GET", gatewayConfigUrl, nil)
	if err != nil {
		return nil, nil, err
	}
	request.Header.Add(main.TokenHeaderKey, token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	var gwConfig bootstrap.GatewayConfig
	err = json.NewDecoder(response.Body).Decode(&gwConfig)
	if err != nil {
		return nil, nil, err
	}

	return &gwConfig, response, err
}

func setup(listener net.Listener) (*http.Server, error) {
	c := map[string]string{"user": "pass"}
	azureAuthMock := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("mock authed")
			next.ServeHTTP(w, r)
		})
	}

	server := &http.Server{Handler: main.Api(c, azureAuthMock)}
	go server.Serve(listener)
	time.Sleep(1 * time.Second)

	response, err := http.DefaultClient.Get(fmt.Sprintf("http://%s%s", listener.Addr().String(), "/isalive"))
	if err != nil {
		return nil, nil
	}
	defer response.Body.Close()
	return server, err
}

func addToken(token string) (*http.Response, error) {
	request, err := http.NewRequest("POST", tokenUrl, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add(main.TokenHeaderKey, token)
	request.SetBasicAuth("user", "pass")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}
