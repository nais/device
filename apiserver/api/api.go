package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nais/device/apiserver/database"
	log "github.com/sirupsen/logrus"
)

type api struct {
	db database.APIServerDB
}

type Peer struct {
	PublicKey string
	IP        string
}

// TODO(jhrv): do actual filtering of the clients.
// TODO(jhrv): keep cache of gateway access group members to remove AAD runtime dependency
// gatewayConfig returns the clients for the gateway that has the group membership required
func (a *api) gatewayConfig(w http.ResponseWriter, r *http.Request) {
	//gateway := chi.URLParam(r, "gateway")

	clients, err := a.db.ReadClients()

	if err != nil {
		log.Errorf("reading clients from database: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	peers := make([]Peer, 0)
	for _, client := range clients {
		peers = append(peers, Peer{PublicKey: client.PublicKey, IP: client.IP})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(peers)
}

func (a *api) clients(w http.ResponseWriter, r *http.Request) {
	clients, err := a.db.ReadClients()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(clients)
}

func (a *api) updateHealth(w http.ResponseWriter, r *http.Request) {
	var healthUpdates []database.Client
	if err := json.NewDecoder(r.Body).Decode(&healthUpdates); err != nil {
		defer r.Body.Close()

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("error during JSON unmarshal: %s\n", err)))
		return
	}

	// Abort status update if it contains incomplete entries
	// is_healthy and serial is required
	for _, s := range healthUpdates {
		if s.Healthy == nil || len(s.Serial) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing required field\n"))
			return
		}
	}

	if err := a.db.UpdateClientStatus(healthUpdates); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		w.Write([]byte("unable to persist client statuses\n"))
		return
	}
}
