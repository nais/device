package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/nais/device/internal/apiserver/kekw"
	"github.com/nais/device/internal/device-agent/agenthttp"
	"github.com/nais/device/internal/device-agent/open"
	"github.com/nais/device/internal/humanerror"
	"github.com/nais/device/internal/random"
	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Handler interface {
	GetDeviceAgentToken(ctx context.Context, log logrus.FieldLogger, oauthConfig oauth2.Config) (*Tokens, error)
}

type handler struct {
	authChannel chan *authFlowResponse
	authServer  string

	// Needs to be set before every autg flow
	state        string
	oauthConfig  oauth2.Config
	codeVerifier *codeverifier.CodeVerifier
}

func (h *handler) valid() error {
	if h.oauthConfig.ClientID == "" || h.oauthConfig.RedirectURL == "" || h.oauthConfig.Endpoint.AuthURL == "" || h.oauthConfig.Endpoint.TokenURL == "" {
		return fmt.Errorf("oauth2 config is missing required fields")
	}
	if h.codeVerifier == nil {
		return fmt.Errorf("code verifier is not set")
	}
	if h.state == "" {
		return fmt.Errorf("state is not set")
	}
	return nil
}

func New(authServer string) *handler {
	h := &handler{
		authChannel: make(chan *authFlowResponse),
		authServer:  authServer,
	}

	// define a handler that will get the authorization code, call the authFlowResponse endpoint, and close the HTTP server
	agenthttp.HandleFunc("GET /", h.handleRedirectAzure)
	agenthttp.HandleFunc("GET /google", h.handleRedirectGoogle)

	return h
}

type authFlowResponse struct {
	Tokens *Tokens
	err    error
}

type Tokens struct {
	Token   *oauth2.Token
	IDToken string
}

func (h *handler) GetDeviceAgentToken(ctx context.Context, log logrus.FieldLogger, oauthConfig oauth2.Config) (*Tokens, error) {
	// Ignoring impossible error
	h.codeVerifier, _ = codeverifier.CreateCodeVerifier()
	h.state = random.RandomString(16, random.LettersAndNumbers)
	h.oauthConfig = oauthConfig

	redirectURL := strings.Replace(agenthttp.Addr(), "127.0.0.1", "localhost", 1)
	h.oauthConfig.RedirectURL = strings.Replace(h.oauthConfig.RedirectURL, "ADDR", redirectURL, 1)

	url := h.oauthConfig.AuthCodeURL(
		h.state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", h.codeVerifier.CodeChallengeS256()))

	open.Open(url)
	log.WithField("url", url).Info("if the browser didn't open, visit url to sign in")

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, humanerror.Wrap(ctx.Err(), "Login process timed out, please restart by connecting again.")
		} else if errors.Is(ctx.Err(), context.Canceled) {
			return nil, humanerror.Wrap(ctx.Err(), "Login process was cancelled, please restart by connecting again.")
		}
		return nil, fmt.Errorf("authFlow: %w", ctx.Err())
	case authFlowResponse := <-h.authChannel:
		if authFlowResponse.err != nil {
			return nil, fmt.Errorf("authFlow: %w", authFlowResponse.err)
		}

		return authFlowResponse.Tokens, nil
	}
}

func failAuth(err error, w http.ResponseWriter, authFlowChan chan *authFlowResponse) {
	failureResponse(w, err.Error())
	authFlowChan <- &authFlowResponse{Tokens: nil, err: err}
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

func successfulResponse(w http.ResponseWriter, msg, userAgent string) {
	content := fmt.Sprintf(`
<div style="position:absolute;left:50%%;top:50%%;margin-top:-150px;margin-left:-200px;height:300px;width:400px;bottom:50%%;background-color:#f5f5f5;border:1px solid #d9d9d9;border-radius:4px">
	<img style="width:100px;display:block;margin:auto;margin-top:50px" width="100" src="data:image/jpeg;base64,%s"/>
	<p style="margin-top: 70px" align="center">
	%s
	</p>
</div>
`, kekw.HappyKekW, msg)

	if isChrome(userAgent) {
		content += `
<div style="position:absolute;bottom:1em;left:1em;">
	<p>Hvis du prøvde å åpne en side før du logget inn på naisdevice, vil ikke Chrome merke det før du har sletta åpne sockets. Dette kan du gjøre med å navigere til:</p><input type="text" readonly="" value="chrome://net-internals#sockets" style="width: 16em;" onfocus="this.select()"><p></p>
</div>`
	}

	w.Header().Set("content-type", "text/html;charset=utf8")
	_, _ = fmt.Fprint(w, content)
}

func isChrome(userAgent string) bool {
	return strings.Contains(userAgent, "Chrome/")
}
