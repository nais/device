package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nais/device/apiserver/jita"
	"github.com/nais/device/pkg/pb"
	"time"

	"net/http"

	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
)

type api struct {
	db   *database.APIServerDB
	jita *jita.Jita
}

const (
	MaxTimeSinceKolideLastSeen = 1 * time.Hour
)

type GatewayConfig struct {
	Devices []database.Device
	Routes  []string
}

// gatewayConfig returns the devices for the gateway that has the group membership required
func (a *api) gatewayConfig(w http.ResponseWriter, r *http.Request) {
	gatewayName, _, _ := r.BasicAuth()

	ctx := context.Background()
	sessionInfos, err := a.db.ReadSessionInfos(ctx)

	if err != nil {
		log.Errorf("reading session infos from database: %v", err)
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
		Devices: healthy(authorized(gateway.AccessGroupIDs, a.privileged(*gateway, sessionInfos))),
		Routes:  gateway.Routes,
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(gatewayConfig)
	if err != nil {
		log.Errorf("writing gateway config response: %v", err)
		return
	}

	m, err := GatewayConfigsReturned.GetMetricWithLabelValues(gateway.Name)
	if err != nil {
		log.Errorf("getting metric metric: %v", err)
	}
	m.Inc()
}

func (api *api) privileged(gateway pb.Gateway, sessions []database.SessionInfo) []database.SessionInfo {
	if !gateway.RequiresPrivilegedAccess {
		return sessions
	}
	privilegedUsers, err := api.jita.GetPrivilegedUsersForGateway(gateway.Name)
	if err != nil {
		log.Errorf("Gateway retrieving privileged users, %s", err)
	}

	m, err := PrivilegedUsersPerGateway.GetMetricWithLabelValues(gateway.Name)
	if err != nil {
		log.Errorf("getting metric metric: %v", err)
	}
	m.Set(float64(len(privilegedUsers)))

	var sessionsToReturn []database.SessionInfo
	for _, session := range sessions {
		if userIsPrivileged(privilegedUsers, session.ObjectId) {
			sessionsToReturn = append(sessionsToReturn, session)
		} else {
			log.Tracef("Skipping unauthorized session: %s", session.Device.Serial)
		}
	}
	return sessionsToReturn
}

func healthy(devices []database.Device) []database.Device {
	var healthyDevices []database.Device
	timeNow := time.Now()
	for _, device := range devices {
		kolideLastSeenDevice := time.Unix(0, 0)
		if device.KolideLastSeen != nil {
			kolideLastSeenDevice = time.Unix(*device.KolideLastSeen, 0)
		}

		if *device.Healthy {
			if timeNow.After(kolideLastSeenDevice.Add(MaxTimeSinceKolideLastSeen)) {
				log.Debugf("Would have skipped device: %s with owner %s. (last seen: %s, now: %s).", device.Serial, device.Username, kolideLastSeenDevice, timeNow)
			}

			healthyDevices = append(healthyDevices, device)
		}

	}

	return healthyDevices
}

func authorized(gatewayGroups []string, sessions []database.SessionInfo) []database.Device {
	var authorizedDevices []database.Device

	for _, session := range sessions {
		if userIsAuthorized(gatewayGroups, session.Groups) {
			authorizedDevices = append(authorizedDevices, *session.Device)
		} else {
			log.Tracef("Skipping unauthorized session: %s", session.Device.Serial)
		}
	}

	return authorizedDevices
}

func (a *api) devices(w http.ResponseWriter, r *http.Request) {
	devices, err := a.db.ReadDevices()

	if err != nil {
		log.Errorf("Reading devices from database: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
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

	if err := a.db.UpdateDevice(healthUpdates); err != nil {
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
		logWithFields.Errorf("Encoding gateways response: %v", err)
		respondf(w, http.StatusInternalServerError, "unable to get device config\n")
		return
	}

	m, err := DeviceConfigsReturned.GetMetricWithLabelValues(device.Serial, device.Username)
	if err != nil {
		logWithFields.Errorf("getting metric metric: %v", err)
	}
	m.Inc()

	logWithFields.Debugf("Successfully returned config to device")
}

func (a *api) UserGateways(userGroups []string) (*[]pb.Gateway, error) {
	gateways, err := a.db.ReadGateways()
	if err != nil {
		return nil, fmt.Errorf("reading gateways from db: %v", err)
	}

	var filtered []pb.Gateway
	for _, gw := range gateways {
		if userIsAuthorized(gw.AccessGroupIDs, userGroups) {
			filtered = append(filtered, gw)
		}
	}

	return &filtered, nil
}

func userIsPrivileged(privilegedUsers []jita.PrivilegedUser, users string) bool {
	for _, privilegedUser := range privilegedUsers {
		if privilegedUser.UserId == users {
			return true
		}
	}
	return false
}

func userIsAuthorized(gatewayGroups []string, userGroups []string) bool {
	for _, userGroup := range userGroups {
		for _, gatewayGroup := range gatewayGroups {
			if userGroup == gatewayGroup {
				return true
			}
		}
	}
	return false
}

func respondf(w http.ResponseWriter, statusCode int, format string, args ...interface{}) {
	w.WriteHeader(statusCode)

	if _, wErr := w.Write([]byte(fmt.Sprintf(format, args...))); wErr != nil {
		log.Errorf("unable to write response: %v", wErr)
	}
}
