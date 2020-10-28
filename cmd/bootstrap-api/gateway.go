package main

import (
	"encoding/json"
	"github.com/nais/device/pkg/bootstrap"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

/*
-> = POST
<- = GET

0. apiserver -> token         -> bootstrap-api
1. gateway   -> GatewayInfo   -> bootstrap-api
2. apiserver <- GatewayInfo   <- bootstrap-api
3. apiserver -> GatewayConfig -> bootstrap-api
4. gateway   <- GatewayConfig <- bootstrap-api
*/


//step 0: apiserver posts token
func postToken(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get(TokenHeaderKey)
	if len(token) == 0 {
		log.WithField("event", "no_token_header").Info("Error getting token during postGatewayConfig")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gatewayEnrollments.addToken(token)
	w.WriteHeader(http.StatusCreated)
}

// step 1. gateway posts gateway info
func postGatewayInfo(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get(TokenHeaderKey)
	if len(token) == 0 {
		log.WithField("event", "no_token_header").Info("Error getting token during postGatewayConfig")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var gatewayInfo bootstrap.GatewayInfo
	err := json.NewDecoder(r.Body).Decode(&gatewayInfo)

	if err != nil {
		log.Errorf("Decoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gatewayEnrollments.addGatewayInfo(gatewayInfo)

	log.WithFields(log.Fields{
		"component": "bootstrap-api",
		"serial":    gatewayInfo.Name,
		"public_ip": gatewayInfo.PublicIP,
	}).Infof("Gateway enrollment request for apiserer queued")

	w.WriteHeader(http.StatusCreated)
}

// step 2: apiserver gets gateway infos
func getGatewayInfo(w http.ResponseWriter, r *http.Request) {
	gatewayInfos := gatewayEnrollments.getGatewayInfos()
	log.Infof("%s %s: %v", r.Method, r.URL, gatewayInfos)

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(gatewayInfos)
	if err != nil {
		log.Errorf("Encoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// step 3. apiserver posts gateway config
func postGatewayConfig(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get(TokenHeaderKey)
	if len(token) == 0 {
		log.WithField("event", "no_token_header").Info("Error getting token during postGatewayConfig")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log := log.WithFields(log.Fields{
		"component": "bootstrap-api",
	})

	var gatewayConfig bootstrap.GatewayConfig
	err := json.NewDecoder(r.Body).Decode(&gatewayConfig)
	if err != nil {
		log.Errorf("Decoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	gatewayEnrollments.addGatewayConfig(token, gatewayConfig)

	w.WriteHeader(http.StatusCreated)

	log.WithField("event", "apiserver_posted_gateway_config").Infof("Successful gateway enrollment response came from apiserver")
}

// step 4. gateway requests gateway config
func getGatewayConfig(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get(TokenHeaderKey)
	if len(token) == 0 {
		log.WithField("event", "no_token_header").Info("Error getting token during getGatewayConfig")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log := log.WithFields(log.Fields{
		"component": "bootstrap-api",
	})

	gatewayConfig := gatewayEnrollments.getGatewayConfig(token)

	if gatewayConfig == nil {
		w.WriteHeader(http.StatusNotFound)
		log.Warnf("no gateway config for provided token found")
		return
	}

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(gatewayConfig)
	if err != nil {
		log.Errorf("Unable to get gateway config: Encoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.WithField("event", "retrieved_gateway_config").Infof("Successfully returned gateway config")
}

type ActiveGatewayEnrollments struct {
	gatewayInfos     []bootstrap.GatewayInfo
	gatewayInfosLock sync.Mutex

	bootstrapGatewayConfigs     map[string]bootstrap.GatewayConfig
	bootstrapGatewayConfigsLock sync.Mutex

	tokens     []string
	tokensLock sync.Mutex
}

func (a *ActiveGatewayEnrollments) init() {
	a.bootstrapGatewayConfigs = make(map[string]bootstrap.GatewayConfig)
}

func (a *ActiveGatewayEnrollments) getGatewayInfos() []bootstrap.GatewayInfo {
	a.gatewayInfosLock.Lock()
	defer a.gatewayInfosLock.Unlock()

	var gatewayInfosToReturn []bootstrap.GatewayInfo
	gatewayInfosToReturn = append(gatewayInfosToReturn, a.gatewayInfos...)

	a.gatewayInfos = nil

	return gatewayInfosToReturn
}

func (a *ActiveGatewayEnrollments) addGatewayInfo(gatewayInfo bootstrap.GatewayInfo) {
	a.gatewayInfosLock.Lock()
	defer a.gatewayInfosLock.Unlock()

	a.gatewayInfos = append(a.gatewayInfos, gatewayInfo)
}

func (a *ActiveGatewayEnrollments) addGatewayConfig(serial string, bootstrapGatewayConfig bootstrap.GatewayConfig) {
	a.bootstrapGatewayConfigsLock.Lock()
	defer a.bootstrapGatewayConfigsLock.Unlock()

	a.bootstrapGatewayConfigs[serial] = bootstrapGatewayConfig
}

func (a *ActiveGatewayEnrollments) getGatewayConfig(serial string) *bootstrap.GatewayConfig {
	a.bootstrapGatewayConfigsLock.Lock()
	defer a.bootstrapGatewayConfigsLock.Unlock()

	if val, ok := a.bootstrapGatewayConfigs[serial]; ok {
		delete(a.bootstrapGatewayConfigs, serial)
		return &val
	} else {
		return nil
	}
}

func (a *ActiveGatewayEnrollments) hasToken(token string) bool {
	a.tokensLock.Lock()
	defer a.tokensLock.Unlock()

	for _, validToken := range a.tokens {
		if validToken == token {
			return true
		}
	}
	return false
}

func (a *ActiveGatewayEnrollments) addToken(token string) {
	a.tokensLock.Lock()
	defer a.tokensLock.Unlock()

	a.tokens = append(a.tokens, token)
}

func (a *ActiveGatewayEnrollments) deleteToken(token string) {
	a.tokensLock.Lock()
	defer a.tokensLock.Unlock()

	for i, validToken := range a.tokens {
		if validToken == token {
			a.tokens = append(a.tokens[:i], a.tokens[i+1:]...)
			return
		}
	}
}
