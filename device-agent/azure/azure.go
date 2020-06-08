package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/nais/device/apiserver/kekw"
	"github.com/nais/device/pkg/random"
	codeverifier "github.com/nirasan/go-oauth-pkce-code-verifier"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const fileName string = "token.jwt"

func EnsureClient(ctx context.Context, conf oauth2.Config, tokenDir string) (*http.Client, error) {
	var token *oauth2.Token
	var err error

	save := func() {
		err = saveToken(*token, tokenDir)
		if err != nil {
			log.Errorf("Unable to store the token %v", err)
		}
	}

	token, err = loadToken(tokenDir)
	if err == nil {
		log.Info("Token loaded from disk")
		src := conf.TokenSource(ctx, token)
		token, err = src.Token()
		if err == nil {
			save()
			return conf.Client(ctx, token), nil
		}
		log.Info("Failed refreshing token")
	}

	log.Info("Unable to use token from disk, fetching a new one")

	if token == nil {
		token, err = runAuthFlow(ctx, conf)
		if err != nil {
			return nil, fmt.Errorf("running authorization code flow: %w", err)
		}
	}

	save()
	return conf.Client(ctx, token), nil
}

func runAuthFlow(ctx context.Context, conf oauth2.Config) (*oauth2.Token, error) {
	server := &http.Server{Addr: "127.0.0.1:51800"}

	// Ignoring impossible error
	codeVerifier, _ := codeverifier.CreateCodeVerifier()

	method := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	challenge := oauth2.SetAuthURLParam("code_challenge", codeVerifier.CodeChallengeS256())

	// TODO check this in response from Azure
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

	go server.ListenAndServe()
	defer server.Close()

	url := conf.AuthCodeURL(randomString, oauth2.AccessTypeOffline, method, challenge)
	err := openDefaultBrowser(url)
	if err != nil {
		log.Errorf("opening browser, err: %v", err)
		// Don't return, as this is not fatal (user can open browser manually)
	}
	fmt.Printf("If the browser didn't open, visit this url to sign in: %v\n", url)

	token := <-tokenChan

	if token == nil {
		return nil, fmt.Errorf("no token received")
	}

	return token, nil
}

func loadToken(tokenDir string) (*oauth2.Token, error) {
	log.Info("Loading token from file")
	tokenBytes, err := ioutil.ReadFile(path.Join(tokenDir, fileName))
	if err != nil {
		return nil, err
	}

	token := oauth2.Token{}
	err = json.Unmarshal(tokenBytes, &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func saveToken(token oauth2.Token, tokenDir string) error {
	jsonToken, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(tokenDir, fileName), jsonToken, os.FileMode(0660))
	if err != nil {
		return err
	}

	return nil
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
