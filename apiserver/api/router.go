package api

import (
	"net/http"

	"github.com/go-chi/chi"
	chi_middleware "github.com/go-chi/chi/middleware"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/apiserver/middleware"
)

type Config struct {
	DB                          *database.APIServerDB
	OAuthKeyValidatorMiddleware func(next http.Handler) http.Handler
	APIKeys                     map[string]string
}

func New(cfg Config) chi.Router {
	api := api{db: cfg.DB}

	latencyHistBuckets := []float64{.001, .005, .01, .025, .05, .1, .5, 1, 3, 5}
	prometheusMiddleware := middleware.PrometheusMiddleware("apiserver", latencyHistBuckets...)
	prometheusMiddleware.Initialize("/devices", http.MethodGet, http.StatusOK)

	r := chi.NewRouter()

	r.Use(prometheusMiddleware.Handler())

	r.Get("/gateways/{gateway}", api.gatewayConfig)
	r.Route("/gateways", func(r chi.Router) {
		if cfg.APIKeys != nil {
			r.Use(chi_middleware.BasicAuth("naisdevice", cfg.APIKeys))
		}
		r.Get("/", api.gateways)
	})
	r.Get("/devices", api.devices)
	r.Put("/devices/health", api.updateHealth)
	r.Route("/devices/config/{serial}", func(r chi.Router) {
		if cfg.OAuthKeyValidatorMiddleware != nil {
			r.Use(cfg.OAuthKeyValidatorMiddleware)
		}
		r.Get("/", api.deviceConfig)
	})

	return r
}
