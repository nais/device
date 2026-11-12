package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/nais/device/internal/deviceagent/agenthttp"
)

func TestHandleRedirectGoogle_RedirectsToAgentPage(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(ExchangeResponse{
			Token:   &oauth2.Token{AccessToken: "access-token"},
			IDToken: "id-token",
		})
	}))
	defer authServer.Close()

	verifier, err := codeverifier.CreateCodeVerifier()
	require.NoError(t, err)

	jitaPage := agenthttp.Path("/jita?gateway=gw", true)
	h := &handler{
		authChannel:  make(chan *authFlowResponse, 1),
		authServer:   authServer.URL,
		state:        "state",
		codeVerifier: verifier,
		redirect:     jitaPage,
		log:          logrus.New(),
		oauthConfig: oauth2.Config{
			ClientID:    "client-id",
			RedirectURL: "http://localhost/auth/google",
			Endpoint:    oauth2.Endpoint{AuthURL: "https://auth", TokenURL: "https://token"},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/google?state=state&code=code", nil)
	rec := httptest.NewRecorder()
	h.handleRedirectGoogle(rec, req)

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, jitaPage, rec.Header().Get("Location"))

	resp := <-h.authChannel
	require.NoError(t, resp.err)
	assert.Equal(t, "id-token", resp.Tokens.IDToken)
}

func TestRedirectsToAgent(t *testing.T) {
	tests := []struct {
		name     string
		redirect string
		want     bool
	}{
		{name: "jita page", redirect: agenthttp.Path("/jita?gateway=gw", true), want: true},
		{name: "console", redirect: "https://console.nav.cloud.nais.io/?naisdevice=connected", want: false},
		{name: "empty", redirect: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handler{redirect: tt.redirect}
			assert.Equal(t, tt.want, h.redirectsToAgent())
		})
	}
}
