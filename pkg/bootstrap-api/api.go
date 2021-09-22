package bootstrap_api

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"

	"github.com/nais/device/pkg/secretmanager"
)

type DeviceApi struct {
	enrollments    *ActiveDeviceEnrollments
	azureValidator func(http.Handler) http.Handler
	apiserverAuth  func(http.Handler) http.Handler
}

type SecretManager interface {
	GetSecrets(map[string]string) ([]*secretmanager.Secret, error)
	DisableSecret(string) error
}

type GatewayApi struct {
	enrollments          *ActiveGatewayEnrollments
	secretManager        SecretManager
	enrollmentTokens     map[string]string
	enrollmentTokensLock *sync.Mutex
	apiserverAuth        func(http.Handler) http.Handler
}

type Api struct {
	gatewayApi *GatewayApi
	deviceApi  *DeviceApi
}

func (api *Api) Router() chi.Router {
	r := chi.NewRouter()

	r.Get("/isalive", func(w http.ResponseWriter, r *http.Request) {
		return
	})

	r.Route("/api/v1", api.deviceApi.RoutesV1())

	r.Route("/api/v2", func(r chi.Router) {
		r.Route("/device", api.deviceApi.RoutesV2())
		r.Route("/gateway", api.gatewayApi.RoutesV2())
	})

	return r
}

func NewApi(apiserverCredentialEntries map[string]string, azureValidator func(next http.Handler) http.Handler, secretManager SecretManager) *Api {
	api := &Api{}

	apiserverAuth := chi_middleware.BasicAuth("naisdevice", apiserverCredentialEntries)

	api.gatewayApi = &GatewayApi{
		enrollments:          NewActiveGatewayEnrollments(),
		secretManager:        secretManager,
		enrollmentTokens:     nil,
		enrollmentTokensLock: &sync.Mutex{},
		apiserverAuth:        apiserverAuth,
	}

	api.deviceApi = &DeviceApi{
		enrollments:    NewActiveDeviceEnrollments(),
		apiserverAuth:  apiserverAuth,
		azureValidator: azureValidator,
	}

	return api
}

func (api *Api) SyncEnrollmentSecretsLoop(syncInterval time.Duration, stop chan struct{}) {
	api.gatewayApi.syncEnrollmentSecretsLoop(syncInterval, stop)
}

func (api *DeviceApi) RoutesV1() func(chi.Router) {
	// state
	return func(r chi.Router) {
		// device calls
		r.Group(func(r chi.Router) {
			r.Use(api.azureValidator)
			r.Post("/deviceinfo", api.postDeviceInfo)
			r.Get("/bootstrapconfig/{serial}", api.getBootstrapConfig)
		})

		// apiserver calls
		r.Group(func(r chi.Router) {
			r.Use(api.apiserverAuth)
			r.Get("/deviceinfo", api.getDeviceInfos)
			r.Post("/bootstrapconfig/{serial}", api.postBootstrapConfig)
		})
	}
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

func (api *GatewayApi) RoutesV2() func(chi.Router) {
	return func(r chi.Router) {
		// gateway calls
		r.Group(func(r chi.Router) {
			r.Use(api.gatewayAuth)
			r.Post("/info", api.postGatewayInfo)
			r.Get("/config/{name}", api.getGatewayConfig)
		})

		// apiserver calls
		r.Group(func(r chi.Router) {
			r.Use(api.apiserverAuth)
			r.Get("/info", api.getGatewayInfo)
			r.Post("/config/{name}", api.postGatewayConfig)
		})
	}
}
