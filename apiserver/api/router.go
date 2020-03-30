package api

import (
	"github.com/go-chi/chi"
	"github.com/nais/device/apiserver/database"
)

type Config struct {
	DB database.APIServerDB
}

func New(cfg Config) chi.Router {
	api := api{db: cfg.DB}

	r := chi.NewRouter()
	r.Get("/gateways/{gateway}", api.gatewayConfig)
	r.Get("/clients", api.clients)
	r.Put("/clients/health", api.updateHealth)

	return r
}
