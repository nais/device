package bootstrap_api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/auth"
)

type DeviceApi struct {
	enrollments    *ActiveDeviceEnrollments
	tokenValidator func(http.Handler) http.Handler
	apiserverAuth  func(http.Handler) http.Handler
	log            *logrus.Entry
}

type Api struct {
	deviceApi *DeviceApi
}

func (api *Api) Router() chi.Router {
	r := chi.NewRouter()

	r.Get("/isalive", func(w http.ResponseWriter, r *http.Request) {})

	r.Route("/api/v2/device", api.deviceApi.RoutesV2())

	return r
}

func NewApi(apiserverCredentialEntries map[string]string, tokenValidator auth.TokenValidator, log *logrus.Entry) *Api {
	api := &Api{}

	apiserverAuth := chi_middleware.BasicAuth("naisdevice", apiserverCredentialEntries)

	api.deviceApi = &DeviceApi{
		enrollments:    NewActiveDeviceEnrollments(),
		apiserverAuth:  apiserverAuth,
		tokenValidator: tokenValidator,
		log:            log.WithField("subcomponent", "device-api"),
	}

	return api
}

func (api *DeviceApi) RoutesV2() func(chi.Router) {
	// state
	return func(r chi.Router) {
		// device calls
		r.Group(func(r chi.Router) {
			r.Use(api.tokenValidator)
			r.Post("/info", api.postDeviceInfo)
			r.Get("/config/{serial}", api.getBootstrapConfig)
		})

		// apiserver calls
		r.Group(func(r chi.Router) {
			r.Use(api.apiserverAuth)
			r.Get("/info", api.getDeviceInfos)
			r.Post("/config/{serial}", api.postBootstrapConfig)
		})
	}
}
