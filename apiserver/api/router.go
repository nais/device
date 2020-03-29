package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/nais/device/apiserver/api/middleware"
	"github.com/nais/device/apiserver/database"
)

type api struct {
	db database.APIServerDB
}

type Config struct {
	DB database.APIServerDB
}

func New(cfg Config) chi.Router {
	api := api{db: cfg.DB}

	r := chi.NewRouter()
	r.With(middleware.RequestLogger())

	r.Get("/gateways/gw0", api.gatewayConfig())

	return r
}

func (a *api) gatewayConfig() func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		clients, err := a.db.ReadClients()

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(clients)
	}
}
