package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	"github.com/sirupsen/logrus"
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
		defer res.Body.Close()

		var exchangeResponse ExchangeResponse
		err = json.NewDecoder(res.Body).Decode(&exchangeResponse)
		if err != nil {
			failAuth(err, w, authFlowChan)
			return
		}

		ret, err := consoleURL(ctx, exchangeResponse.IDToken, "connected")
		if err != nil {
			logrus.Println("Failed to get console	URL: " + err.Error())
			successfulResponse(w, "Successfully authenticated 👌 Close me pls", r.Header.Get("user-agent"))
		} else {
			http.Redirect(w, r, ret, http.StatusSeeOther)
		}

		tokens := &Tokens{Token: exchangeResponse.Token, IDToken: exchangeResponse.IDToken}
		authFlowChan <- &authFlowResponse{Tokens: tokens, err: nil}
	}
}

func consoleURL(ctx context.Context, idToken, state string) (string, error) {
	// Parse id token to get domain
	t, err := jwt.ParseString(idToken, jwt.WithValidate(false))
	if err != nil {
		return "", err
	}
	hd, _ := t.Get("hd")
	domain, _ := hd.(string)

	if domain == "" {
		return "", fmt.Errorf("could not find domain in id token")
	}

	url := fmt.Sprintf("https://storage.googleapis.com/nais-tenant-data/%s.json", domain)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	d := struct {
		ConsoleURL string `json:"consoleUrl"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&d)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s?naisdevice=%s", d.ConsoleURL, state), nil
}
