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

// TODO(jhrv): do actual filtering of the devices.
// TODO(jhrv): keep cache of gateway access group members to remove AAD runtime dependency
// gatewayConfig returns the devices for the gateway that has the group membership required
func (a *api) gatewayConfig(w http.ResponseWriter, r *http.Request) {
	//gateway := chi.URLParam(r, "gateway")

	devices, err := a.db.ReadDevices()

	if err != nil {
		log.Errorf("reading devices from database: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(healthy(devices))
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

func respondf(w http.ResponseWriter, statusCode int, format string, args ...interface{}) {
	w.WriteHeader(statusCode)

	if _, wErr := w.Write([]byte(fmt.Sprintf(format, args...))); wErr != nil {
		log.Errorf("unable to write response: %v", wErr)
	}
}
