package bootstrap_api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/bootstrap"
)

func (api *DeviceApi) postBootstrapConfig(w http.ResponseWriter, r *http.Request) {
	serial := chi.URLParam(r, "serial")

	ctxLog := api.log.WithField("serial", serial)

	var bootstrapConfig bootstrap.Config
	err := json.NewDecoder(r.Body).Decode(&bootstrapConfig)
	if err != nil {
		ctxLog.Errorf("Decoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	api.enrollments.addBootstrapConfig(serial, bootstrapConfig)

	w.WriteHeader(http.StatusCreated)

	ctxLog.WithField("event", "apiserver_posted_bootstrapconfig").Infof("Successful enrollment response came from apiserver")
}

func (api *DeviceApi) getBootstrapConfig(w http.ResponseWriter, r *http.Request) {
	serial := chi.URLParam(r, "serial")
	ctxLog := api.log.WithFields(log.Fields{
		"serial":   serial,
		"username": r.Context().Value("preferred_username").(string),
	})

	bootstrapConfig := api.enrollments.getBootstrapConfig(serial)

	if bootstrapConfig == nil {
		w.WriteHeader(http.StatusNotFound)
		ctxLog.Warnf("no bootstrap config for serial: %v", serial)
		return
	}

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(bootstrapConfig)
	if err != nil {
		ctxLog.Errorf("Unable to get bootstrap config: Encoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctxLog.WithField("event", "retrieved_bootstrapconfig").Infof("Successfully returned bootstrap config")
}

func (api *DeviceApi) postDeviceInfo(w http.ResponseWriter, r *http.Request) {
	var deviceInfo bootstrap.DeviceInfo
	err := json.NewDecoder(r.Body).Decode(&deviceInfo)

	if err != nil {
		api.log.Errorf("Decoding json: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	deviceInfo.Owner = r.Context().Value("preferred_username").(string)
	if len(deviceInfo.Owner) == 0 {
		api.log.Errorf("deviceInfo without owner, abort enroll")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	api.enrollments.addDeviceInfo(deviceInfo)

	api.log.WithFields(log.Fields{
		"serial":   deviceInfo.Serial,
		"username": deviceInfo.Owner,
		"platform": deviceInfo.Platform,
	}).Infof("Enrollment request for apiserver queued")

	w.WriteHeader(http.StatusCreated)
}

func (api *DeviceApi) getDeviceInfos(w http.ResponseWriter, r *http.Request) {
	deviceInfos := api.enrollments.getDeviceInfos()
	api.log.Infof("%s %s: %v", r.Method, r.URL, deviceInfos)

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(deviceInfos)
	if err != nil {
		api.log.Errorf("Encoding json: %v", err)
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
