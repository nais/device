package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
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
	r.Get("/gateways/{gateway}", api.gatewayConfig)

	return r
}

// TODO(jhrv): do actual filtering of the clients.
// TODO(jhrv): keep cache of gateway access group members to remove AAD runtime dependency
// gatewayConfig returns the clients for the gateway that has the group membership required
func (a *api) gatewayConfig(w http.ResponseWriter, r *http.Request) {
	//gateway := chi.URLParam(r, "gateway")

	clients, err := a.db.ReadClients()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(clients)
}
