package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
)

type api struct {
	db *database.APIServerDB
}

type GatewayConfig struct {
	Devices []database.Device
	Routes  []string
}

// TODO(jhrv): do actual filtering of the devices.
// TODO(jhrv): keep cache of gateway access group members to remove AAD runtime dependency
// gatewayConfig returns the devices for the gateway that has the group membership required
func (a *api) gatewayConfig(w http.ResponseWriter, r *http.Request) {
	gatewayName, _, _ := r.BasicAuth()

	devices, err := a.db.ReadDevices()

	if err != nil {
		log.Errorf("reading devices from database: %v", err)
		respondf(w, http.StatusInternalServerError, "failed getting gateway config")
		return
	}

	gateway, err := a.db.ReadGateway(gatewayName)
	if err != nil {
		log.Errorf("reading gateway from database: %v", err)
		respondf(w, http.StatusInternalServerError, "failed getting gateway config")
		return
	}

	gatewayConfig := GatewayConfig{
		Devices: healthy(devices),
		Routes:  gateway.Routes,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(gatewayConfig)
}

func healthy(devices []database.Device) []database.Device {
	var healthyDevices []database.Device
	for _, device := range devices {
		if *device.Healthy {
			healthyDevices = append(healthyDevices, device)
		} else {
			log.Tracef("Skipping unhealthy device: %s", device.Serial)
		}
	}

	return healthyDevices
}

func (a *api) devices(w http.ResponseWriter, r *http.Request) {
	devices, err := a.db.ReadDevices()

	if err != nil {
		log.Errorf("Reading devices from database: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(devices)
}

func (a *api) updateHealth(w http.ResponseWriter, r *http.Request) {
	var healthUpdates []database.Device
	if err := json.NewDecoder(r.Body).Decode(&healthUpdates); err != nil {
		defer r.Body.Close()

		respondf(w, http.StatusBadRequest, "error during JSON unmarshal: %s\n", err)
		return
	}

	// Abort status update if it contains incomplete entries
	// is_healthy and serial is required
	for _, s := range healthUpdates {
		if s.Healthy == nil || len(s.Serial) == 0 {
			respondf(w, http.StatusBadRequest, "missing required field\n")
			return
		}
	}

	if err := a.db.UpdateDeviceStatus(healthUpdates); err != nil {
		log.Error(err)
		respondf(w, http.StatusInternalServerError, "unable to persist device statuses\n")
		return
	}
}

func (a *api) gateways(w http.ResponseWriter, r *http.Request) {
	//serial := chi.URLParam(r, "serial")
	gateways, err := a.db.ReadGateways()
	if err != nil {
		log.Errorf("reading gateways: %v", err)
		respondf(w, http.StatusInternalServerError, "unable to get device config\n")
		return
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(gateways)

	if err != nil {
		log.Errorf("encoding gateways response: %v", err)
		respondf(w, http.StatusInternalServerError, "unable to get device config\n")
		return
	}
}

func (a *api) deviceConfig(w http.ResponseWriter, r *http.Request) {
	sessionInfo := r.Context().Value("sessionInfo").(*database.SessionInfo)

	log := log.WithFields(log.Fields{
		"username":  sessionInfo.Device.Username,
		"serial":    sessionInfo.Device.Serial,
		"platform":  sessionInfo.Device.Platform,
		"component": "apiserver",
	})

	// Don't reuse Device from Session here as it might be outdated.
	device, err := a.db.ReadDeviceById(r.Context(), sessionInfo.Device.ID)
	if err != nil {
		log.Errorf("Reading device from db: %v", err)
		respondf(w, http.StatusInternalServerError, "error reading device from db")
		return
	}

	if !*device.Healthy {
		log.Infof("Device is unhealthy, returning HTTP %v", http.StatusForbidden)
		respondf(w, http.StatusForbidden, "device not healthy, on slack: /msg @Kolide status")
		return
	}

	gateways, err := a.UserGateways(sessionInfo.Groups)

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(gateways)

	if err != nil {
		log.Errorf("Encoding gateways response: %v", err)
		respondf(w, http.StatusInternalServerError, "unable to get device config\n")
		return
	}

	log.Infof("Successfully returned config to device")
}

func (a *api) UserGateways(userGroups []string) (*[]database.Gateway, error) {
	gateways, err := a.db.ReadGateways()
	if err != nil {
		return nil, fmt.Errorf("reading gateways from db: %v", err)
	}

	userIsAuthorized := func(gatewayGroups []string, userGroups []string) bool {
		for _, userGroup := range userGroups {
			for _, gatewayGroup := range gatewayGroups {
				if userGroup == gatewayGroup {
					return true
				}
			}
		}
		return false
	}

	var filtered []database.Gateway
	for _, gw := range gateways {
		if userIsAuthorized(gw.AccessGroupIDs, userGroups) {
			filtered = append(filtered, gw)
		}
	}

	return &filtered, nil
}

func respondf(w http.ResponseWriter, statusCode int, format string, args ...interface{}) {
	w.WriteHeader(statusCode)

	if _, wErr := w.Write([]byte(fmt.Sprintf(format, args...))); wErr != nil {
		log.Errorf("unable to write response: %v", wErr)
	}
}
