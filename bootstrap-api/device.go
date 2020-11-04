package bootstrap_api

import (
	"encoding/json"
	"github.com/go-chi/chi"
	"github.com/nais/device/pkg/bootstrap"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

func (api *DeviceApi) postBootstrapConfig(w http.ResponseWriter, r *http.Request) {
	serial := chi.URLParam(r, "serial")

	log := log.WithFields(log.Fields{
		"component": "bootstrap-api",
		"serial":    serial,
	})

	var bootstrapConfig bootstrap.Config
	err := json.NewDecoder(r.Body).Decode(&bootstrapConfig)
	if err != nil {
		log.Errorf("Decoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	api.enrollments.addBootstrapConfig(serial, bootstrapConfig)

	w.WriteHeader(http.StatusCreated)

	log.WithField("event", "apiserver_posted_bootstrapconfig").Infof("Successful enrollment response came from apiserver")
}

func (api *DeviceApi) getBootstrapConfig(w http.ResponseWriter, r *http.Request) {
	serial := chi.URLParam(r, "serial")
	log := log.WithFields(log.Fields{
		"component": "bootstrap-api",
		"serial":    serial,
		"username":  r.Context().Value("preferred_username").(string),
	})

	bootstrapConfig := api.enrollments.getBootstrapConfig(serial)

	if bootstrapConfig == nil {
		w.WriteHeader(http.StatusNotFound)
		log.Warnf("no bootstrap config for serial: %v", serial)
		return
	}

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(bootstrapConfig)
	if err != nil {
		log.Errorf("Unable to get bootstrap config: Encoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.WithField("event", "retrieved_bootstrapconfig").Infof("Successfully returned bootstrap config")
}

func (api *DeviceApi) postDeviceInfo(w http.ResponseWriter, r *http.Request) {
	var deviceInfo bootstrap.DeviceInfo
	err := json.NewDecoder(r.Body).Decode(&deviceInfo)

	if err != nil {
		log.Errorf("Decoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	deviceInfo.Owner = r.Context().Value("preferred_username").(string)
	if len(deviceInfo.Owner) == 0 {
		log.Errorf("deviceInfo without owner, abort enroll")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	api.enrollments.addDeviceInfo(deviceInfo)

	log.WithFields(log.Fields{
		"component": "bootstrap-api",
		"serial":    deviceInfo.Serial,
		"username":  deviceInfo.Owner,
		"platform":  deviceInfo.Platform,
	}).Infof("Enrollment request for apiserver queued")

	w.WriteHeader(http.StatusCreated)
}

func (api *DeviceApi) getDeviceInfos(w http.ResponseWriter, r *http.Request) {
	deviceInfos := api.enrollments.getDeviceInfos()
	log.Infof("%s %s: %v", r.Method, r.URL, deviceInfos)

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(deviceInfos)
	if err != nil {
		log.Errorf("Encoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type ActiveDeviceEnrollments struct {
	deviceInfos     []bootstrap.DeviceInfo
	deviceInfosLock sync.Mutex

	bootstrapConfigs     map[string]bootstrap.Config
	bootstrapConfigsLock sync.Mutex
}

func NewActiveDeviceEnrollments() *ActiveDeviceEnrollments {
	return &ActiveDeviceEnrollments{
		bootstrapConfigs: make(map[string]bootstrap.Config),
	}
}

func (a *ActiveDeviceEnrollments) getDeviceInfos() []bootstrap.DeviceInfo {
	a.deviceInfosLock.Lock()
	defer a.deviceInfosLock.Unlock()

	var deviceInfosToReturn []bootstrap.DeviceInfo
	deviceInfosToReturn = append(deviceInfosToReturn, a.deviceInfos...)

	a.deviceInfos = nil

	return deviceInfosToReturn
}

func (a *ActiveDeviceEnrollments) addDeviceInfo(deviceInfo bootstrap.DeviceInfo) {
	a.deviceInfosLock.Lock()
	defer a.deviceInfosLock.Unlock()

	a.deviceInfos = append(a.deviceInfos, deviceInfo)
}

func (a *ActiveDeviceEnrollments) addBootstrapConfig(serial string, bootstrapConfig bootstrap.Config) {
	a.bootstrapConfigsLock.Lock()
	defer a.bootstrapConfigsLock.Unlock()

	a.bootstrapConfigs[serial] = bootstrapConfig
}

func (a *ActiveDeviceEnrollments) getBootstrapConfig(serial string) *bootstrap.Config {
	a.bootstrapConfigsLock.Lock()
	defer a.bootstrapConfigsLock.Unlock()

	if val, ok := a.bootstrapConfigs[serial]; ok {
		delete(a.bootstrapConfigs, serial)
		return &val
	} else {
		return nil
	}
}
