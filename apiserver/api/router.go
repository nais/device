package api

import (
	"context"
	"github.com/go-chi/chi"
	chi_middleware "github.com/go-chi/chi/middleware"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/apiserver/middleware"
	"github.com/nais/device/apiserver/session"
	"net/http"
)

type Config struct {
	DB       *database.APIServerDB
	APIKeys  map[string]string
	Sessions *session.Sessions
}

func New(ctx context.Context, cfg Config) chi.Router {
	api := api{db: cfg.DB}
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
		r.Get("/gateways/{gateway}/devices", api.gatewayConfig)
		r.Get("/devices", api.devices)
		r.Put("/devices/health", api.updateHealth)
	})

	r.Route("/devices/{serial}/gateways", func(r chi.Router) {
		r.Use(sessions.Validator(ctx))
		r.Get("/", api.deviceConfig)
	})

	r.Post("/login", sessions.Login)

	return r
}
