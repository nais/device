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

type ClientRegistrationRequest struct {
	Username  string
	PublicKey string
	Serial    string
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
		if *client.Healthy {
			peers = append(peers, Peer{PublicKey: client.PublicKey, IP: client.IP})
		} else {
			log.Tracef("Skipping unhealthy client: %s", client.Serial)
		}
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

		respondf(w, http.StatusBadRequest, "error during JSON unmarshal: %s\n", err)
		return
	}

	// Abort status update if it contains incomplete entries
	// is_healthy and serial is required
	for _, s := range healthUpdates {
		if s.Healthy == nil || len(s.Serial) == 0 {
			respondf(w, http.StatusBadRequest, "missing required field\n")
			return
		}
	}

	if err := a.db.UpdateClientStatus(healthUpdates); err != nil {
		log.Error(err)
		respondf(w, http.StatusInternalServerError, "unable to persist client statuses\n")
		return
	}
}

func (a *api) registerClient(w http.ResponseWriter, r *http.Request) {
	var reg ClientRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		respondf(w, http.StatusBadRequest, "error during JSON unmarshal: %s\n", err)
	}

	if err := a.db.AddClient(reg.Username, reg.PublicKey, reg.Serial); err != nil {
		respondf(w, http.StatusInternalServerError, "unable to add new peer: %s\n", err)
	}
}

func respondf(w http.ResponseWriter, statusCode int, format string, args ...interface{}) {
	w.WriteHeader(statusCode)

	if _, wErr := w.Write([]byte(fmt.Sprintf(format, args...))); wErr != nil {
		log.Errorf("unable to write client response: %v", wErr)
	}
}
