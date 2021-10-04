package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/nais/device/pkg/apiserver/kekw"
	"github.com/nais/device/pkg/azure"
	"github.com/nais/device/pkg/device-agent/open"
	"github.com/nais/device/pkg/random"
)

type authFlowResponse struct {
	Token *oauth2.Token
	err   error
}

func GetDeviceAgentToken(ctx context.Context, conf *oauth2.Config) (*oauth2.Token, error) {
	// Ignoring impossible error
	codeVerifier, _ := codeverifier.CreateCodeVerifier()

	authFlowChan := make(chan *authFlowResponse)
	handler := http.NewServeMux()
	state := random.RandomString(16, random.LettersAndNumbers)

	// define a handler that will get the authorization code, call the authFlowResponse endpoint, and close the HTTP server
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(30*time.Second))
		defer cancel()

		codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier.String())
		t, err := conf.Exchange(ctx, code, codeVerifierParam)
		if err != nil {
			failAuth(fmt.Errorf("exchanging code for tokens: %w", err), w, authFlowChan)
			return
		}

		parsedToken, err := jwt.Parse([]byte(t.AccessToken))
		if err != nil {
			failAuth(fmt.Errorf("parsing authFlowResponse: %v", err), w, authFlowChan)
			return
		}

		groups, ok := parsedToken.Get("groups")
		if !ok {
			failAuth(fmt.Errorf("no groups found in authFlowResponse"), w, authFlowChan)
			return
		}

		approvalOK := false
		for _, group := range groups.([]interface{}) {
			if group.(string) == azure.NaisDeviceApprovalGroup {
				approvalOK = true
			}
		}

		if !approvalOK {
			http.Redirect(w, r, "https://naisdevice-approval.nais.io/", http.StatusSeeOther)
			authFlowChan <- &authFlowResponse{Token: nil, err: fmt.Errorf("do's and don'ts not accepted, opening https://naisdevice-approval.nais.io/ in browser")}
			return
		}

		successfulResponse(w, "Successfully authenticated 👌 Close me pls")
		authFlowChan <- &authFlowResponse{Token: t, err: nil}
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")

	if err != nil {
		return nil, fmt.Errorf("creating listener: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	conf.RedirectURL = fmt.Sprintf("http://localhost:%d/", port)

	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	defer server.Close()

	url := conf.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", codeVerifier.CodeChallengeS256()))

	err = open.Open(url)
	if err != nil {
		log.Errorf("opening browser, err: %v", err)
		// Don't return, as this is not fatal (user can open browser manually)
	}
	fmt.Printf("If the browser didn't open, visit this url to sign in: %v\n", url)

	authFlowResponse := <-authFlowChan

	if authFlowResponse.err != nil {
		return nil, fmt.Errorf("authFlow: %w", authFlowResponse.err)
	}

	return authFlowResponse.Token, nil
}

func failAuth(err error, w http.ResponseWriter, authFlowChan chan *authFlowResponse) {
	failureResponse(w, err.Error())
	authFlowChan <- &authFlowResponse{Token: nil, err: err}
}

func failureResponse(w http.ResponseWriter, msg string) {
	w.Header().Set("content-type", "text/html;charset=utf8")
	_, _ = fmt.Fprintf(w, `
<div style="position:absolute;left:50%%;top:50%%;margin-top:-150px;margin-left:-200px;height:300px;width:400px;bottom:50%%;background-color:#f5f5f5;border:1px solid #d9d9d9;border-radius:4px">
<img style="width:100px;display:block;margin:auto;margin-top:50px" width="100" src="data:image/jpeg;base64,%s"/>
<p style="margin-top: 70px" align="center">
  %s
</p>
</div>
`, kekw.SadKekW, msg)
}

func successfulResponse(w http.ResponseWriter, msg string) {
	w.Header().Set("content-type", "text/html;charset=utf8")
	_, _ = fmt.Fprintf(w, `
<div style="position:absolute;left:50%%;top:50%%;margin-top:-150px;margin-left:-200px;height:300px;width:400px;bottom:50%%;background-color:#f5f5f5;border:1px solid #d9d9d9;border-radius:4px">
<img style="width:100px;display:block;margin:auto;margin-top:50px" width="100" src="data:image/jpeg;base64,%s"/>
<p style="margin-top: 70px" align="center">
  %s
</p>
</div>
`, kekw.HappyKekW, msg)
}
