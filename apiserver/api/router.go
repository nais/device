package api

import (
	"github.com/go-chi/chi"
	chi_middleware "github.com/go-chi/chi/middleware"
	"github.com/nais/device/apiserver/auth"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/apiserver/jita"
	"github.com/nais/device/apiserver/middleware"
	"net/http"
)

type Config struct {
	DB       *database.APIServerDB
	Jita     *jita.Jita
	APIKeys  map[string]string
	Sessions *auth.Sessions
}

func New(cfg Config) chi.Router {
	api := api{db: cfg.DB, jita: cfg.Jita}
	sessions := cfg.Sessions

	latencyHistBuckets := []float64{.001, .005, .01, .025, .05, .1, .5, 1, 3, 5}
	prometheusMiddleware := middleware.PrometheusMiddleware("apiserver", latencyHistBuckets...)
	prometheusMiddleware.Initialize("/devices", http.MethodGet, http.StatusOK)

	r := chi.NewRouter()

	r.Use(prometheusMiddleware.Handler())

	r.Group(func(r chi.Router) {
		if cfg.APIKeys != nil {
			r.Use(chi_middleware.BasicAuth("naisdevice", cfg.APIKeys))
		}

		r.Get("/gateways", api.gateways)
		r.Get("/devices", api.devices)

		r.Get("/gatewayconfig", api.gatewayConfig)
	})

	r.Group(func(r chi.Router) {
		r.Use(sessions.Validator())
		r.Get("/deviceconfig", api.deviceConfig)
	})

	r.Get("/login", sessions.Login)
	r.Get("/authurl", sessions.AuthURL)

	return r
}
