package jita

import (
	"cmp"
	"embed"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/nais/device/internal/deviceagent/agenthttp"
	"github.com/nais/device/internal/deviceagent/html"
	"github.com/nais/device/internal/deviceagent/runtimeconfig"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:embed jita.html
var template embed.FS

type Handler struct {
	log logrus.FieldLogger
	rc  runtimeconfig.RuntimeConfig
}

func redirectToIndexWithErrorMessage(message string, w http.ResponseWriter, req *http.Request) {
	gateway := url.QueryEscape(req.FormValue("gateway"))
	message = url.QueryEscape(message)
	path := fmt.Sprintf("/jita?gateway=%s&errorMessage=%s", gateway, message)
	http.Redirect(w, req, agenthttp.Path(path, true), http.StatusSeeOther)
}

func redirectToIndexWithStatusMessage(message string, w http.ResponseWriter, req *http.Request) {
	gateway := url.QueryEscape(req.FormValue("gateway"))
	message = url.QueryEscape(message)
	path := fmt.Sprintf("/jita?gateway=%s&statusMessage=%s", gateway, message)
	http.Redirect(w, req, agenthttp.Path(path, true), http.StatusSeeOther)
}

func (h *Handler) index(w http.ResponseWriter, req *http.Request) {
	gateway := req.URL.Query().Get("gateway")
	if gateway == "" {
		http.Error(w, "Missing gateway parameter.", http.StatusBadRequest)
		return
	}

	var hasActiveRequest bool
	var grants []*pb.GatewayJitaGrant
	var apiserverError string
	if err := h.rc.WithAPIServer(func(apiserver pb.APIServerClient, key string) error {
		hasAccessResp, err := apiserver.UserHasAccessToPrivilegedGateway(req.Context(), &pb.UserHasAccessToPrivilegedGatewayRequest{
			SessionKey: key,
			Gateway:    gateway,
		})
		if err != nil {
			return err
		}

		grantsResp, err := apiserver.GetGatewayJitaGrantsForUser(req.Context(), &pb.GetGatewayJitaGrantsForUserRequest{
			SessionKey: key,
		})
		if err != nil {
			return err
		}

		hasActiveRequest = hasAccessResp.HasAccess
		grants = grantsResp.GatewayJitaGrants

		return nil
	}); err != nil {
		h.log.WithError(err).Errorf("unable to communicate with apiserver")
		h.verifyToken(err)
		apiserverError = apiserverErrorMessage(err)
	}

	type accessGrant struct {
		Created    time.Time
		Expires    time.Time
		Revoked    *time.Time
		Gateway    string
		Reason     string
		IsRevoked  bool
		HasExpired bool
	}

	data := struct {
		GrantGatewayAccessRequestFormAction string
		RevokeGatewayAccessFormAction       string
		Gateway                             string
		HasActiveAccessRequest              bool
		Grants                              []accessGrant
		ErrorMessage                        string
		StatusMessage                       string
		ApiserverAvailable                  bool
	}{
		GrantGatewayAccessRequestFormAction: agenthttp.Path("/jita/grantGatewayAccessRequest", true),
		RevokeGatewayAccessFormAction:       agenthttp.Path("/jita/revokeGatewayAccess", true),
		Gateway:                             gateway,
		HasActiveAccessRequest:              hasActiveRequest,
		Grants: func(grants []*pb.GatewayJitaGrant) []accessGrant {
			ret := make([]accessGrant, len(grants))
			for i, grant := range grants {
				var revoked *time.Time
				expires := grant.Expires.AsTime().Local()
				if grant.Revoked.IsValid() {
					r := grant.Revoked.AsTime().Local()
					revoked = &r
				}
				ret[i] = accessGrant{
					Created:    grant.Created.AsTime().Local(),
					Expires:    expires,
					Revoked:    revoked,
					Gateway:    grant.Gateway,
					Reason:     grant.Reason,
					IsRevoked:  grant.Revoked != nil,
					HasExpired: expires.Before(time.Now()),
				}
			}
			return ret
		}(grants),
		ErrorMessage:       cmp.Or(apiserverError, req.URL.Query().Get("errorMessage")),
		StatusMessage:      req.URL.Query().Get("statusMessage"),
		ApiserverAvailable: apiserverError == "",
	}

	if err := html.Render(w, template, "jita.html", data); err != nil {
		h.log.WithError(err).Error("rendering page")
		http.Error(w, "Failed to render page.", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) verifyToken(err error) {
	if status.Convert(err).Code() == codes.Unauthenticated {
		h.rc.SetJitaToken(nil)
	}
}

// apiserverErrorMessage maps gRPC status codes to user-facing error messages.
func apiserverErrorMessage(err error) string {
	switch status.Convert(err).Code() {
	case codes.Unauthenticated:
		return "Your login session or JITA token is no longer valid. Please reconnect Naisdevice and try again."
	case codes.Unavailable, codes.DeadlineExceeded:
		return "Unable to reach the apiserver. Check that Naisdevice is connected, and try again."
	default:
		return "Unable to communicate with apiserver. If the problem persists, please send your logs to the Nais team."
	}
}

func (h *Handler) grant(w http.ResponseWriter, req *http.Request) {
	gateway := req.FormValue("gateway")
	if gateway == "" {
		http.Error(w, "Missing gateway parameter.", http.StatusBadRequest)
		return
	}

	reason := req.FormValue("reason")
	if reason == "" {
		redirectToIndexWithErrorMessage("Missing reason parameter.", w, req)
		return
	}

	duration, err := strconv.Atoi(req.FormValue("duration"))
	if err != nil {
		redirectToIndexWithErrorMessage("Missing or invalid duration parameter.", w, req)
		return
	} else if duration < 1 || duration > 8 {
		redirectToIndexWithErrorMessage("Invalid duration parameter, must be between 1 and 8, inclusive.", w, req)
		return
	}

	token := h.rc.GetJitaToken(req.Context())
	if token == nil {
		http.Error(w, "No token found. Make sure you open this page using the systray / cli.", http.StatusUnauthorized)
		return
	}

	if err := h.rc.WithAPIServer(func(apiserver pb.APIServerClient, key string) error {
		_, err := apiserver.GrantPrivilegedGatewayAccess(req.Context(), &pb.GrantPrivilegedGatewayAccessRequest{
			SessionKey: key,
			Token:      token.AccessToken,
			NewPrivilegedGatewayAccess: &pb.NewPrivilegedGatewayAccess{
				Gateway: gateway,
				Expires: timestamppb.New(time.Now().Add(time.Hour * time.Duration(duration))),
				Reason:  reason,
			},
		})
		return err
	}); err != nil {
		h.log.WithError(err).Errorf("unable to communicate with apiserver")
		h.verifyToken(err)
		redirectToIndexWithErrorMessage(apiserverErrorMessage(err), w, req)
		return
	}

	redirectToIndexWithStatusMessage("You have been granted access. The gateway will connect shortly.", w, req)
}

func (h *Handler) revoke(w http.ResponseWriter, req *http.Request) {
	gatewayToRevoke := req.FormValue("gatewayToRevoke")
	if gatewayToRevoke == "" {
		redirectToIndexWithErrorMessage("Missing gatewayToRevoke parameter.", w, req)
		return
	}

	if err := h.rc.WithAPIServer(func(apiserver pb.APIServerClient, key string) error {
		_, err := apiserver.RevokePrivilegedGatewayAccess(req.Context(), &pb.RevokePrivilegedGatewayAccessRequest{
			SessionKey: key,
			Gateway:    gatewayToRevoke,
		})
		return err
	}); err != nil {
		h.log.WithError(err).Errorf("unable to communicate with apiserver")
		h.verifyToken(err)
		redirectToIndexWithErrorMessage(apiserverErrorMessage(err), w, req)
		return
	}

	redirectToIndexWithStatusMessage("The access to the gateway has been revoked.", w, req)
}

func New(rc runtimeconfig.RuntimeConfig, log logrus.FieldLogger) *Handler {
	return &Handler{
		rc:  rc,
		log: log,
	}
}

// Register registers the Jita handler routes using the provided registerFunc.
func (h *Handler) Register(registerFunc func(pattern string, handler http.HandlerFunc)) {
	registerFunc("GET /jita", h.index)
	registerFunc("POST /jita/grantGatewayAccessRequest", h.grant)
	registerFunc("POST /jita/revokeGatewayAccess", h.revoke)
}
