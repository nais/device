package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/jita"
	apiserver_metrics "github.com/nais/device/pkg/apiserver/metrics"
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

func (a *api) deviceConfig(w http.ResponseWriter, r *http.Request) {
	sessionInfo := auth.GetSessionInfo(r.Context())

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
		if StringSliceHasIntersect(gw.AccessGroupIDs, userGroups) {
			gw.PasswordHash = ""
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

// Returns true if any of the strings in one of the slices are found in the other slice; otherwise false.
func StringSliceHasIntersect(slice1 []string, slice2 []string) bool {
	for _, a := range slice1 {
		for _, b := range slice2 {
			if a == b {
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
