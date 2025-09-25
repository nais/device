package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/nais/device/internal/apiserver/kekw"
	"github.com/nais/device/internal/device-agent/open"
	"github.com/nais/device/internal/humanerror"
	"github.com/nais/device/internal/random"
	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type GetTokenFunc func(context.Context, logrus.FieldLogger, oauth2.Config, string) (*Tokens, error)

type authFlowResponse struct {
	Tokens *Tokens
	err    error
}

type Tokens struct {
	Token   *oauth2.Token
	IDToken string
}

func GetDeviceAgentToken(ctx context.Context, log logrus.FieldLogger, conf oauth2.Config, authServer string) (*Tokens, error) {
	// Ignoring impossible error
	codeVerifier, _ := codeverifier.CreateCodeVerifier()

	authFlowChan := make(chan *authFlowResponse)
	state := random.RandomString(16, random.LettersAndNumbers)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("creating listener: %w", err)
	}

	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	conf.RedirectURL = strings.Replace(conf.RedirectURL, "PORT", port, 1)

	handler := http.NewServeMux()
	// define a handler that will get the authorization code, call the authFlowResponse endpoint, and close the HTTP server
	handler.HandleFunc("GET /", handleRedirectAzure(state, conf, codeVerifier, authFlowChan))
	handler.HandleFunc("GET /google", handleRedirectGoogle(state, conf.RedirectURL, codeVerifier, authFlowChan, authServer))

	server := &http.Server{Handler: handler}
	go server.Serve(listener)
	defer server.Close()

	url := conf.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", codeVerifier.CodeChallengeS256()))

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
	case authFlowResponse := <-authFlowChan:
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
