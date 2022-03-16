package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	"net/http"
	"time"
)

type ExchangeRequest struct {
	CodeVerifier string `json:"code_verifier"`
	AccessCode   string `json:"access_code"`
	RedirectURI  string `json:"redirect_uri"`
}

type ExchangeResponse struct {
	IDToken string    `json:"access_token"`
	Expiry  time.Time `json:"expiry"`
}

func handleRedirectGoogle(state, redirectURI string, codeVerifier *codeverifier.CodeVerifier, authFlowChan chan *authFlowResponse) http.HandlerFunc {
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

		ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(10*time.Second))
		defer cancel()

		exchangeRequest := ExchangeRequest{
			AccessCode:   code,
			CodeVerifier: codeVerifier.String(),
			RedirectURI:  redirectURI,
		}
		buffer := bytes.Buffer{}
		err := json.NewEncoder(&buffer).Encode(exchangeRequest)
		if err != nil {
			failAuth(err, w, authFlowChan)
			return
		}

		//req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://naisdevice-auth-server-h2pjqrstja-lz.a.run.app/exchange", &buffer)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:8080/exchange", &buffer)
		if err != nil {
			failAuth(err, w, authFlowChan)
			return
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			failAuth(err, w, authFlowChan)
			return
		}

		var exchangeResponse ExchangeResponse
		err = json.NewDecoder(res.Body).Decode(&exchangeResponse)
		if err != nil {
			failAuth(err, w, authFlowChan)
			return
		}

		token := &Token{
			Token:  exchangeResponse.IDToken,
			Expiry: exchangeResponse.Expiry,
		}

		successfulResponse(w, "Successfully authenticated 👌 Close me pls")
		authFlowChan <- &authFlowResponse{Token: token, err: nil}
	}
}
