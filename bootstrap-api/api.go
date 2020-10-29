package bootstrap_api

import (
	"github.com/go-chi/chi"
	chi_middleware "github.com/go-chi/chi/middleware"
	"net/http"
)

func Api(apiserverCredentialEntries map[string]string, azureValidator func(next http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	r.Get("/isalive", func(w http.ResponseWriter, r *http.Request) {
		return
	})

	r.Route("/api/v1", func(r chi.Router) {
		// device calls
		r.Group(func(r chi.Router) {
			r.Use(azureValidator)
			r.Post("/deviceinfo", postDeviceInfo)
			r.Get("/bootstrapconfig/{serial}", getBootstrapConfig)
		})

		// apiserver calls
		r.Group(func(r chi.Router) {
			r.Use(chi_middleware.BasicAuth("naisdevice", apiserverCredentialEntries))
			r.Get("/deviceinfo", getDeviceInfos)
			r.Post("/bootstrapconfig/{serial}", postBootstrapConfig)
		})

		// gateway calls
		r.Group(func(r chi.Router) {
			r.Use(TokenAuth)
			r.Post("/gatewayinfo", postGatewayInfo)
			r.Get("/gatewayconfig", getGatewayConfig)
		})

		// apiserver calls
		r.Group(func(r chi.Router) {
			r.Use(chi_middleware.BasicAuth("naisdevice", apiserverCredentialEntries))
			r.Get("/gatewayinfo", getGatewayInfo)
			r.Post("/gatewayconfig", postGatewayConfig)
			r.Post("/token", postToken)
		})
	})

	return r
}
