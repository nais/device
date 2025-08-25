package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/oauth2"
)

func handleRedirectAzure(state string, conf oauth2.Config, codeVerifier *codeverifier.CodeVerifier, authFlowChan chan *authFlowResponse) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Catch if user has not approved terms
		responseState := r.URL.Query().Get("state")
		if state != responseState {
			failAuth(fmt.Errorf("invalid 'state' in auth response, try again"), w, authFlowChan)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			failAuth(fmt.Errorf("could not find 'code' URL query parameter"), w, authFlowChan)
			return
		}

		ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(30*time.Second))
		defer cancel()

		codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier.String())
		t, err := conf.Exchange(ctx, code, codeVerifierParam)
		if err != nil {
			failAuth(fmt.Errorf("exchanging code for tokens: %w", err), w, authFlowChan)
			return
		}

		http.Redirect(w, r, "https://console.nav.cloud.nais.io/?naisdevice=connected", http.StatusSeeOther)

		authFlowChan <- &authFlowResponse{Tokens: &Tokens{Token: t}, err: nil}
	}
}
