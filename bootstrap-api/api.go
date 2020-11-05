package bootstrap_api

import (
	"github.com/go-chi/chi"
	chi_middleware "github.com/go-chi/chi/middleware"
	"github.com/nais/device/pkg/secretmanager"
	"net/http"
	"sync"
	"time"
)

type DeviceApi struct {
	enrollments    *ActiveDeviceEnrollments
	azureValidator func(http.Handler) http.Handler
	apiserverAuth  func(http.Handler) http.Handler
}

type SecretManager interface {
	ListSecrets(map[string]string) ([]*secretmanager.Secret, error)
}

type GatewayApi struct {
	enrollments          *ActiveGatewayEnrollments
	secretManager        SecretManager
	enrollmentTokens     map[string]string
	enrollmentTokensLock *sync.Mutex
	apiserverAuth        func(http.Handler) http.Handler
}

func NewApi(apiserverCredentialEntries map[string]string, azureValidator func(next http.Handler) http.Handler, secretManager SecretManager, syncInterval time.Duration) chi.Router {
	r := chi.NewRouter()

	r.Get("/isalive", func(w http.ResponseWriter, r *http.Request) {
		return
	})

	apiserverAuth := chi_middleware.BasicAuth("naisdevice", apiserverCredentialEntries)

	gatewayApi := GatewayApi{
		enrollments:          NewActiveGatewayEnrollments(),
		secretManager:        secretManager,
		enrollmentTokens:     nil,
		enrollmentTokensLock: &sync.Mutex{},
		apiserverAuth:        apiserverAuth,
	}

	/*  TODO use this in real world
	    stop := make(chan struct{})
		go gatewayApi.syncEnrollmentSecretsLoop(syncInterval, stop)
	*/
	gatewayApi.syncEnrollmentSecrets()
	deviceApi := DeviceApi{
		enrollments:    NewActiveDeviceEnrollments(),
		apiserverAuth:  apiserverAuth,
		azureValidator: azureValidator,
	}

	r.Route("/api/v1", deviceApi.RoutesV1())

	r.Route("/api/v2", func(r chi.Router) {
		r.Route("/device", deviceApi.RoutesV2())
		r.Route("/gateway", gatewayApi.RoutesV2())
	})

	return r
}

func (api *DeviceApi) RoutesV1() func(chi.Router) {
	//state
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
	//state
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
			r.Get("/config", api.getGatewayConfig)
		})

		// apiserver calls
		r.Group(func(r chi.Router) {
			r.Use(api.apiserverAuth)
			r.Get("/info", api.getGatewayInfo)
			r.Post("/config", api.postGatewayConfig)
		})
	}
}
