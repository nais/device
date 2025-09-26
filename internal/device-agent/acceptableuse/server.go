package acceptableuse

import (
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/nais/device/internal/device-agent/agenthttp"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	log logrus.FieldLogger
	rc  runtimeconfig.RuntimeConfig
}

func (h *Handler) index(w http.ResponseWriter, req *http.Request) {
	var acceptedAt time.Time
	if err := h.rc.WithAPIServer(func(apiserver pb.APIServerClient, key string) error {
		if resp, err := apiserver.GetAcceptableUseAcceptedAt(req.Context(), &pb.GetAcceptableUseAcceptedAtRequest{
			SessionKey: key,
		}); err != nil {
			return err
		} else if resp.AcceptedAt != nil {
			acceptedAt = resp.AcceptedAt.AsTime()
		}
		return nil
	}); err != nil {
		h.log.Errorf("while getting acceptable use acceptance: %s", err)
		http.Error(w, "Failed to get acceptable use acceptance.", http.StatusInternalServerError)
		return
	}

	data := struct {
		HasAccepted bool
		AcceptedAt  string
		FormAction  string
	}{
		HasAccepted: !acceptedAt.IsZero(),
		AcceptedAt:  acceptedAt.Local().Format(time.RFC822),
		FormAction:  agenthttp.Path("/acceptableUse", true),
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

	if err := h.rc.WithAPIServer(func(apiserver pb.APIServerClient, key string) error {
		_, err := apiserver.SetAcceptableUseAccepted(req.Context(), &pb.SetAcceptableUseAcceptedRequest{
			SessionKey: key,
			Accepted:   accepted,
		})
		return err
	}); err != nil {
		h.log.Errorf("while setting acceptable use acceptance: %s", err)
		http.Error(w, "Failed to set acceptable use acceptance.", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, agenthttp.Path("/acceptableUse", true), http.StatusSeeOther)
}

func New(rc runtimeconfig.RuntimeConfig, log logrus.FieldLogger) *Handler {
	handler := &Handler{
		rc:  rc,
		log: log,
	}

	return handler
}

// Register registers the acceptable use handler routes using the provided registerFunc.
func (h *Handler) Register(registerFunc func(pattern string, handler http.HandlerFunc)) {
	registerFunc("GET /acceptableUse", h.index)
	registerFunc("POST /acceptableUse", h.set)
}
