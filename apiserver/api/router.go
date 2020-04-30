package api

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/nais/device/apiserver/database"
)

type Config struct {
	DB                          *database.APIServerDB
	OAuthKeyValidatorMiddleware func(next http.Handler) http.Handler
}

func New(cfg Config) chi.Router {
	api := api{db: cfg.DB}

	r := chi.NewRouter()
	r.Get("/gateways/{gateway}", api.gatewayConfig)
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
