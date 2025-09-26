package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
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

func (h *handler) handleRedirectGoogle(w http.ResponseWriter, r *http.Request) {
	if err := h.valid(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Catch if user has not approved terms
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

	ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(10*time.Second))
	defer cancel()

	exchangeRequest := ExchangeRequest{
		AccessCode:   code,
		CodeVerifier: h.codeVerifier.String(),
		RedirectURI:  h.oauthConfig.RedirectURL,
	}
	buffer := bytes.Buffer{}
	err := json.NewEncoder(&buffer).Encode(exchangeRequest)
	if err != nil {
		failAuth(err, w, h.authChannel)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.authServer+"/exchange", &buffer)
	if err != nil {
		failAuth(err, w, h.authChannel)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		failAuth(err, w, h.authChannel)
		return
	}
	defer res.Body.Close()

	var exchangeResponse ExchangeResponse
	err = json.NewDecoder(res.Body).Decode(&exchangeResponse)
	if err != nil {
		failAuth(err, w, h.authChannel)
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
	h.authChannel <- &authFlowResponse{Tokens: tokens, err: nil}
}

func consoleURL(ctx context.Context, idToken, state string) (string, error) {
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
	fmt.Println("Fetching console URL from ", url)
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
