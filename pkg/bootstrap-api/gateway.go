package bootstrap_api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/bootstrap"
	"github.com/nais/device/pkg/version"
)

/*
-> = POST
<- = GET

1. gateway   -> GatewayInfo   -> bootstrap-api
2. apiserver <- GatewayInfo   <- bootstrap-api
3. apiserver -> GatewayConfig -> bootstrap-api
4. gateway   <- GatewayConfig <- bootstrap-api
*/

const GatewayNameContextKey = "gateway-name"

var failedSecretManagerSynchronizations prometheus.Counter

func init() {
	failedSecretManagerSynchronizations = prometheus.NewCounter(prometheus.CounterOpts{
		Name:        "failed_secret_manager_synchronizations",
		Help:        "count of failed secret manager synchronizations",
		Namespace:   "naisdevice",
		Subsystem:   "bootstrap_api",
		ConstLabels: prometheus.Labels{"name": "bootstrap-api", "version": version.Version},
	})
}

// step 1. gateway posts gateway info
func (api *GatewayApi) postGatewayInfo(w http.ResponseWriter, r *http.Request) {
	var gatewayInfo bootstrap.GatewayInfo
	err := json.NewDecoder(r.Body).Decode(&gatewayInfo)

	if err != nil {
		log.Errorf("Decoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	api.enrollments.addGatewayInfo(gatewayInfo)

	log.WithFields(log.Fields{
		"component": "bootstrap-api",
		"serial":    gatewayInfo.Name,
		"public_ip": gatewayInfo.PublicIP,
	}).Infof("Gateway enrollment request for apiserver queued")

	w.WriteHeader(http.StatusCreated)
}

// step 2: apiserver gets gateway infos
func (api *GatewayApi) getGatewayInfo(w http.ResponseWriter, r *http.Request) {
	gatewayInfos := api.enrollments.getGatewayInfos()
	log.Infof("%s %s: %v", r.Method, r.URL, gatewayInfos)

	err := json.NewEncoder(w).Encode(gatewayInfos)
	if err != nil {
		log.Errorf("Encoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// step 3. apiserver posts gateway config
func (api *GatewayApi) postGatewayConfig(w http.ResponseWriter, r *http.Request) {
	gatewayName := chi.URLParam(r, "name")

	log := log.WithFields(log.Fields{
		"component": "bootstrap-api",
	})

	var gatewayConfig bootstrap.Config
	err := json.NewDecoder(r.Body).Decode(&gatewayConfig)
	if err != nil {
		log.Errorf("Decoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	api.enrollments.addGatewayConfig(gatewayConfig, gatewayName)

	w.WriteHeader(http.StatusCreated)

	log.WithField("event", "apiserver_posted_gateway_config").Infof("Successful gateway enrollment response came from apiserver")
}

// step 4. gateway requests gateway config
func (api *GatewayApi) getGatewayConfig(w http.ResponseWriter, r *http.Request) {
	gatewayName := chi.URLParam(r, "name")

	log := log.WithFields(log.Fields{
		"component": "bootstrap-api",
	})

	gatewayConfig := api.enrollments.getGatewayConfig(gatewayName)

	if gatewayConfig == nil {
		w.WriteHeader(http.StatusNotFound)
		log.Warnf("No gateway config for provided token found")
		return
	}

	if err := json.NewEncoder(w).Encode(gatewayConfig); err != nil {
		log.Errorf("Encoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	secretName := fmt.Sprintf("enrollment-token_%s", gatewayName)
	if err := api.secretManager.DisableSecret(secretName); err != nil {
		log.Errorf("Disabling secret: %s: %v", secretName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.WithField("event", "retrieved_gateway_config").Infof("Successfully returned gateway config")
}

type ActiveGatewayEnrollments struct {
	gatewayInfos     []bootstrap.GatewayInfo
	gatewayInfosLock sync.Mutex

	bootstrapGatewayConfigs     map[string]bootstrap.Config
	bootstrapGatewayConfigsLock sync.Mutex
}

func NewActiveGatewayEnrollments() *ActiveGatewayEnrollments {
	return &ActiveGatewayEnrollments{
		bootstrapGatewayConfigs: make(map[string]bootstrap.Config),
	}
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

func (a *ActiveGatewayEnrollments) addGatewayConfig(bootstrapGatewayConfig bootstrap.Config, name string) {
	a.bootstrapGatewayConfigsLock.Lock()
	defer a.bootstrapGatewayConfigsLock.Unlock()

	a.bootstrapGatewayConfigs[name] = bootstrapGatewayConfig
}

func (a *ActiveGatewayEnrollments) getGatewayConfig(gatewayName string) *bootstrap.Config {
	a.bootstrapGatewayConfigsLock.Lock()
	defer a.bootstrapGatewayConfigsLock.Unlock()

	if val, ok := a.bootstrapGatewayConfigs[gatewayName]; ok {
		delete(a.bootstrapGatewayConfigs, gatewayName)
		return &val
	} else {
		return nil
	}
}

func (api *GatewayApi) authenticated(providedGatewayName, providedToken string) bool {
	api.enrollmentTokensLock.Lock()
	defer api.enrollmentTokensLock.Unlock()

	gatewayName, ok := api.enrollmentTokens[providedToken]
	if !ok {
		log.Debugf("auth token not found for gateway: %s", providedGatewayName)
		return false
	}

	return strings.HasSuffix(gatewayName, providedGatewayName)
}

func (api *GatewayApi) syncEnrollmentSecretsLoop(interval time.Duration, stop chan struct{}) {
	for {
		select {
		case <-time.After(interval):
			api.syncEnrollmentSecrets()
		case <-stop:
			return
		}
	}
}

func (api *GatewayApi) syncEnrollmentSecrets() {
	api.enrollmentTokensLock.Lock()
	defer api.enrollmentTokensLock.Unlock()

	filter := map[string]string{"type": "enrollment-token"}
	secrets, err := api.secretManager.GetSecrets(filter)
	if err != nil {
		log.Errorf("Listing secrets: %v", err)
		failedSecretManagerSynchronizations.Inc()
		return
	}

	api.enrollmentTokens = make(map[string]string)
	for _, secret := range secrets {
		api.enrollmentTokens[string(secret.Data)] = secret.Name
	}
}

func (api *GatewayApi) deleteToken(name string) {
	api.enrollmentTokensLock.Lock()
	defer api.enrollmentTokensLock.Unlock()

	delete(api.enrollmentTokens, name)
}
