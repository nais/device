package bootstrap_api

import (
	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
	"net/http"

	"github.com/nais/device/pkg/secretmanager"
)

type DeviceApi struct {
	enrollments    *ActiveDeviceEnrollments
	azureValidator func(http.Handler) http.Handler
	apiserverAuth  func(http.Handler) http.Handler
	log            *logrus.Entry
}

type SecretManager interface {
	GetSecrets(map[string]string) ([]*secretmanager.Secret, error)
	DisableSecret(string) error
}

type Api struct {
	deviceApi *DeviceApi
}

type TokenValidator func(next http.Handler) http.Handler

func (api *Api) Router() chi.Router {
	r := chi.NewRouter()

	r.Get("/isalive", func(w http.ResponseWriter, r *http.Request) {
		return
	})

	r.Route("/api/v2/device", api.deviceApi.RoutesV2())

	return r
}

func NewApi(apiserverCredentialEntries map[string]string, azureValidator TokenValidator, log *logrus.Entry) *Api {
	api := &Api{}

	apiserverAuth := chi_middleware.BasicAuth("naisdevice", apiserverCredentialEntries)

	api.deviceApi = &DeviceApi{
		enrollments:    NewActiveDeviceEnrollments(),
		apiserverAuth:  apiserverAuth,
		azureValidator: azureValidator,
		log:            log.WithField("subcomponent", "device-api"),
	}

	return api
}

func (api *DeviceApi) RoutesV2() func(chi.Router) {
	// state
	return func(r chi.Router) {
		// device calls
		r.Group(func(r chi.Router) {
			r.Use(api.azureValidator)
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
