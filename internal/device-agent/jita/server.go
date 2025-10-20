package jita

import (
	"html/template"
	"net/http"

	"github.com/nais/device/internal/device-agent/agenthttp"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	log logrus.FieldLogger
	rc  runtimeconfig.RuntimeConfig
}

func (h *Handler) index(w http.ResponseWriter, req *http.Request) {
	data := struct{}{}

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

func (h *Handler) request(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, agenthttp.Path("/jita", true), http.StatusSeeOther)
}

func New(rc runtimeconfig.RuntimeConfig, log logrus.FieldLogger) *Handler {
	handler := &Handler{
		rc:  rc,
		log: log,
	}

	return handler
}

// Register registers the Jita handler routes using the provided registerFunc.
func (h *Handler) Register(registerFunc func(pattern string, handler http.HandlerFunc)) {
	registerFunc("GET /jita", h.index)
	registerFunc("POST /jita", h.request)
}
