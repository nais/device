package acceptableuse

import (
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/nais/device/internal/device-agent/agenthttp"
)

type Handler struct {
	setAcceptance chan<- bool
	acceptedAt    time.Time
}

func (h *Handler) index(w http.ResponseWriter, req *http.Request) {
	data := struct {
		HasAccepted bool
		AcceptedAt  string
		FormAction  string
	}{
		HasAccepted: !h.acceptedAt.IsZero(),
		AcceptedAt:  h.acceptedAt.Local().Format(time.RFC822),
		FormAction:  "/acceptableUse/set?s=" + agenthttp.Secret(),
	}

	t, err := template.ParseFS(templates, "templates/site.html", "templates/index.html")
	if err != nil {
		http.Error(w, "Failed to parse templates.", http.StatusInternalServerError)
		return
	}

	if err := t.ExecuteTemplate(w, "site.html", data); err != nil {
		http.Error(w, "Failed to render index page.", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) set(w http.ResponseWriter, req *http.Request) {
	accepted, err := strconv.ParseBool(req.FormValue("accepted"))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// TODO select with timeout and handle if the channel is blocked
	h.setAcceptance <- accepted

	t, err := template.ParseFS(templates, "templates/site.html", "templates/message.html")
	if err != nil {
		http.Error(w, "Failed to parse templates.", http.StatusInternalServerError)
		return
	}

	if err := t.ExecuteTemplate(w, "site.html", nil); err != nil {
		http.Error(w, "Failed to render index page.", http.StatusInternalServerError)
		return
	}
}

func New(setAcceptance chan<- bool) *Handler {
	handler := &Handler{
		setAcceptance: setAcceptance,
	}

	return handler
}

func (h *Handler) SetAcceptedAt(acceptedAt time.Time) {
	h.acceptedAt = acceptedAt
}

// Register registers the acceptable use handler routes using the provided registerFunc.
func (h *Handler) Register(registerFunc func(pattern string, handler http.HandlerFunc)) {
	registerFunc("GET /acceptableUse", h.index)
	registerFunc("POST /acceptableUse/set", h.set)
}
