package pubsubenroll

import (
	"encoding/json"
	"net/http"

	"github.com/nais/device/pkg/auth"
	log "github.com/sirupsen/logrus"
)

type Handler struct {
	log    *log.Entry
	worker Worker
}

func NewHandler(worker Worker, log *log.Entry) *Handler {
	return &Handler{
		log:    log,
		worker: worker,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var req DeviceRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Errorf("error decoding device request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Owner = auth.GetEmail(r.Context())

	resp, err := h.worker.Send(r.Context(), &req)
	if err != nil {
		log.Errorf("error sending device request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Errorf("error decoding device response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
