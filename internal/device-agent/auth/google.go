package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	"golang.org/x/oauth2"
)

type ExchangeRequest struct {
	CodeVerifier string `json:"code_verifier"`
	AccessCode   string `json:"access_code"`
	RedirectURI  string `json:"redirect_uri"`
}

type ExchangeResponse struct {
	Token   *oauth2.Token `json:"token"`
	IDToken string        `json:"id_token"`
}

func handleRedirectGoogle(state, redirectURI string, codeVerifier *codeverifier.CodeVerifier, authFlowChan chan *authFlowResponse, authServer string) http.HandlerFunc {
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

		// We used to use r.Context() here, but a Google Chrome update broke that.
		// It seems that Chrome closes the HTTP connection prematurely, because the context
		// is at this point already canceled.
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
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

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, authServer+"/exchange", &buffer)
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

		successfulResponse(w, "Successfully authenticated 👌 Close me pls", r.Header.Get("user-agent"))
		tokens := &Tokens{Token: exchangeResponse.Token, IDToken: exchangeResponse.IDToken}
		authFlowChan <- &authFlowResponse{Tokens: tokens, err: nil}
	}
}
