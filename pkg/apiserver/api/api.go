package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nais/device/pkg/apiserver/jita"
	"github.com/nais/device/pkg/apiserver/metrics"
	"github.com/nais/device/pkg/pb"

	"github.com/nais/device/pkg/apiserver/database"
	log "github.com/sirupsen/logrus"
)

type api struct {
	db   database.APIServer
	jita jita.Client
}

const (
	MaxTimeSinceKolideLastSeen = 1 * time.Hour
)

type GatewayConfig struct {
	Devices []*pb.Device
	Routes  []string
}

// gatewayConfig returns the devices for the gateway that has the group membership required
func (a *api) gatewayConfig(w http.ResponseWriter, r *http.Request) {
	gatewayName, _, _ := r.BasicAuth()

	sessionInfos, err := a.db.ReadSessionInfos(r.Context())

	if err != nil {
		log.Errorf("reading session infos from database: %v", err)
		respondf(w, http.StatusInternalServerError, "failed getting gateway config")
		return
	}

	gateway, err := a.db.ReadGateway(r.Context(), gatewayName)
	if err != nil {
		log.Errorf("reading gateway from database: %v", err)
		respondf(w, http.StatusInternalServerError, "failed getting gateway config")
		return
	}

	gatewayConfig := GatewayConfig{
		Devices: healthy(authorized(gateway.AccessGroupIDs, privileged(a.jita, gateway, sessionInfos))),
		Routes:  gateway.Routes,
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(gatewayConfig)
	if err != nil {
		log.Errorf("writing gateway config response: %v", err)
		return
	}

	m, err := apiserver_metrics.GatewayConfigsReturned.GetMetricWithLabelValues(gateway.Name)
	if err != nil {
		log.Errorf("getting metric metric: %v", err)
	}
	m.Inc()
}

func privileged(jita jita.Client, gateway *pb.Gateway, sessions []*pb.Session) []*pb.Session {
	if !gateway.RequiresPrivilegedAccess {
		return sessions
	}
	privilegedUsers, err := jita.GetPrivilegedUsersForGateway(gateway.Name)
	if err != nil {
		log.Errorf("Gateway retrieving privileged users, %s", err)
	}

	m, err := apiserver_metrics.PrivilegedUsersPerGateway.GetMetricWithLabelValues(gateway.Name)
	if err != nil {
		log.Errorf("getting metric metric: %v", err)
	}
	m.Set(float64(len(privilegedUsers)))

	var sessionsToReturn []*pb.Session
	for _, session := range sessions {
		if userIsPrivileged(privilegedUsers, session.ObjectID) {
			sessionsToReturn = append(sessionsToReturn, session)
		} else {
			log.Tracef("Skipping unauthorized session: %s", session.Device.Serial)
		}
	}
	return sessionsToReturn
}

func healthy(devices []*pb.Device) []*pb.Device {
	var healthyDevices []*pb.Device
	timeNow := time.Now()
	for _, device := range devices {
		kolideLastSeenDevice := device.GetKolideLastSeen().AsTime()

		if device.GetHealthy() {
			if timeNow.After(kolideLastSeenDevice.Add(MaxTimeSinceKolideLastSeen)) {
				log.Debugf("Would have skipped device: %s with owner %s. (last seen: %s, now: %s).", device.Serial, device.Username, kolideLastSeenDevice, timeNow)
			}

			healthyDevices = append(healthyDevices, device)
		}

	}

	return healthyDevices
}

func authorized(gatewayGroups []string, sessions []*pb.Session) []*pb.Device {
	var authorizedDevices []*pb.Device

	for _, session := range sessions {
		if userIsAuthorized(gatewayGroups, session.Groups) {
			authorizedDevices = append(authorizedDevices, session.Device)
		} else {
			log.Tracef("Skipping unauthorized session: %s", session.Device.Serial)
		}
	}

	return authorizedDevices
}

func (a *api) devices(w http.ResponseWriter, r *http.Request) {
	devices, err := a.db.ReadDevices(r.Context())

	if err != nil {
		log.Errorf("Reading devices from database: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(devices)
	if err != nil {
		log.Errorf("writing devices response: %v", err)
		return
	}
}

func (a *api) gateways(w http.ResponseWriter, r *http.Request) {
	//serial := chi.URLParam(r, "serial")
	gateways, err := a.db.ReadGateways(r.Context())
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
	sessionInfo := r.Context().Value("sessionInfo").(*pb.Session)

	logWithFields := log.WithFields(log.Fields{
		"username":  sessionInfo.Device.Username,
		"serial":    sessionInfo.Device.Serial,
		"platform":  sessionInfo.Device.Platform,
		"component": "apiserver",
	})

	// Don't reuse Device from Session here as it might be outdated.
	device, err := a.db.ReadDeviceById(r.Context(), sessionInfo.GetDevice().GetId())
	if err != nil {
		logWithFields.Errorf("Reading device from db: %v", err)
		respondf(w, http.StatusInternalServerError, "error reading device from db")
		return
	}

	if !device.GetHealthy() {
		logWithFields.Infof("Device is unhealthy, returning HTTP %v", http.StatusForbidden)
		respondf(w, http.StatusForbidden, "device not healthy, on slack: /msg @Kolide status")
		return
	}

	gateways, err := a.UserGateways(r.Context(), sessionInfo.Groups)

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(gateways)

	if err != nil {
		logWithFields.Errorf("Encoding gateways response: %v", err)
		respondf(w, http.StatusInternalServerError, "unable to get device config\n")
		return
	}

	m, err := apiserver_metrics.DeviceConfigsReturned.GetMetricWithLabelValues(device.Serial, device.Username)
	if err != nil {
		logWithFields.Errorf("getting metric metric: %v", err)
	}
	m.Inc()

	logWithFields.Debugf("Successfully returned config to device")
}

func (a *api) UserGateways(ctx context.Context, userGroups []string) ([]*pb.Gateway, error) {
	gateways, err := a.db.ReadGateways(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading gateways from db: %v", err)
	}

	var filtered []*pb.Gateway
	for _, gw := range gateways {
		if userIsAuthorized(gw.AccessGroupIDs, userGroups) {
			filtered = append(filtered, gw)
		}
	}

	return filtered, nil
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
