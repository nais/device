package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

func (h *handler) handleRedirectAzure(w http.ResponseWriter, r *http.Request) {
	if err := h.valid(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responseState := r.URL.Query().Get("state")
	if h.state != responseState {
		failAuth(fmt.Errorf("invalid 'state' in auth response, try again"), w, h.authChannel)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		failAuth(fmt.Errorf("could not find 'code' URL query parameter"), w, h.authChannel)
		return
	}

	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(30*time.Second))
	defer cancel()

	codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", h.codeVerifier.String())
	t, err := h.oauthConfig.Exchange(ctx, code, codeVerifierParam)
	if err != nil {
		failAuth(fmt.Errorf("exchanging code for tokens: %w", err), w, h.authChannel)
		return
	}

	http.Redirect(w, r, h.redirect, http.StatusSeeOther)

	h.authChannel <- &authFlowResponse{Tokens: &Tokens{Token: t}, err: nil}
}
