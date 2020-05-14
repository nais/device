package azure

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/nais/device/apiserver/kekw"
	"github.com/nais/device/pkg/random"
	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func RunAuthFlow(ctx context.Context, conf oauth2.Config) (*oauth2.Token, error) {
	server := &http.Server{Addr: "127.0.0.1:51800"}

	// Ignoring impossible error
	codeVerifier, _ := codeverifier.CreateCodeVerifier()

	method := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	challenge := oauth2.SetAuthURLParam("code_challenge", codeVerifier.CodeChallengeS256())

	//TODO check this in response from Azure
	randomString := random.RandomString(16, random.LettersAndNumbers)

	tokenChan := make(chan *oauth2.Token)
	// define a handler that will get the authorization code, call the token endpoint, and close the HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			log.Errorf("Error: could not find 'code' URL query parameter")
			failureResponse(w, "Error: could not find 'code' URL query parameter")
			tokenChan <- nil
			return
		}

		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(30*time.Second))
		defer cancel()

		codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier.String())
		t, err := conf.Exchange(ctx, code, codeVerifierParam)
		if err != nil {
			failureResponse(w, "Error: during code exchange")
			log.Errorf("exchanging code for tokens: %v", err)
			tokenChan <- nil
			return
		}

		successfulResponse(w, "Successfully authenticated 👌 Close me pls")
		tokenChan <- t
	})

	go func() {
		_ = server.ListenAndServe()
	}()

	url := conf.AuthCodeURL(randomString, oauth2.AccessTypeOffline, method, challenge)
	command := exec.Command("open", url)
	_ = command.Start()
	fmt.Printf("If the browser didn't open, visit this url to sign in: %v\n", url)

	token := <-tokenChan
	_ = server.Close()

	if token == nil {
		return nil, fmt.Errorf("no token received")
	}

	return token, nil
}

func failureResponse(w http.ResponseWriter, msg string) {
	w.Header().Set("content-type", "text/html;charset=utf8")
	_, _ = fmt.Fprintf(w, `
<h2>
  %s
</h2>
<img width="100" src="data:image/jpeg;base64,%s"/>
`, msg, kekw.SadKekW)
}

func successfulResponse(w http.ResponseWriter, msg string) {
	w.Header().Set("content-type", "text/html;charset=utf8")
	_, _ = fmt.Fprintf(w, `
<h2>
  %s
</h2>
<img width="100" src="data:image/jpeg;base64,%s"/>
`, msg, kekw.HappyKekW)
}
