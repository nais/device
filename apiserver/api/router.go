package api

import (
	"github.com/go-chi/chi"
	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type Config struct {
	DB                          *database.APIServerDB
	OAuthKeyValidatorMiddleware func(next http.Handler) http.Handler
}

func New(cfg Config) chi.Router {
	api := api{db: cfg.DB}

	if cfg.OAuthKeyValidatorMiddleware == nil {
		log.Fatal("Refusing to set up api without token validator")
		return nil
	}

	r := chi.NewRouter()
	r.Get("/gateways/{gateway}", api.gatewayConfig)
	r.Get("/devices", api.devices)
	r.Put("/devices/health", api.updateHealth)
	r.Route("/devices/config/{serial}", func(r chi.Router) {
		r.Use(cfg.OAuthKeyValidatorMiddleware)
		r.Get("/", api.deviceConfig)
	})

	return r
}
