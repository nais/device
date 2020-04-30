package api

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/apiserver/middleware"
)

type Config struct {
	DB                          *database.APIServerDB
	OAuthKeyValidatorMiddleware func(next http.Handler) http.Handler
}

func New(cfg Config) chi.Router {
	api := api{db: cfg.DB}

	latencyHistBuckets := []float64{.001, .005, .01, .025, .05, .1, .5, 1, 3, 5}
	promMiddleware := middleware.PrometheusMiddleware("apiserver", latencyHistBuckets...)
	promMiddleware.Initialize("/devices", http.MethodGet, http.StatusOK)

	r := chi.NewRouter()

	r.Use(promMiddleware.Handler())

	r.Get("/gateways/{gateway}", api.gatewayConfig)
	r.Get("/gateways", api.gateways)
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
