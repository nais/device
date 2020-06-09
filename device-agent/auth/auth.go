package auth

import (
	"context"
	"fmt"
	"github.com/nais/device/apiserver/kekw"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type SessionID string

func RunFlow(ctx context.Context, authURL, apiserverURL string) (SessionID, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	handler := http.NewServeMux()

	sessionIDChan := make(chan string)
	// define a handler that will get the authorization code, call the sessionID endpoint, and close the HTTP server
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			log.Errorf("Error: could not find 'code' URL query parameter")
			failureResponse(w, "Error: could not find 'code' URL query parameter")
			sessionIDChan <- ""
			return
		}

		// post apiserver/login med code
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiserverURL+"/login", strings.NewReader(code))
		if err != nil {
			log.Errorf("Creating post request for apiserver login: %v", err)
			sessionIDChan <- ""
			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Errorf("Posting auth code to apiserver login: %v", err)
			sessionIDChan <- ""
			return
		}

		sessionID, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Reading sessionID from body: %v", err)
			sessionIDChan <- ""
			return
		}

		successfulResponse(w, "Successfully authenticated 👌 Close me pls")
		sessionIDChan <- string(sessionID)
	})

	server := &http.Server{Addr: "127.0.0.1:51800", Handler: handler}
	go server.ListenAndServe()
	defer server.Close()

	err := openDefaultBrowser(authURL)
	if err != nil {
		log.Errorf("opening browser, err: %v", err)
		// Don't return, as this is not fatal (user can open browser manually)
	}
	fmt.Printf("If the browser didn't open, visit this url to sign in: %v\n", authURL)

	var sessionID string
	select {
	case sessionID = <-sessionIDChan:
		break
	case <-time.After(3 * time.Minute):
		log.Warn("timed out waiting for authentication flow")
		break
	}

	if len(sessionID) == 0 {
		return "", fmt.Errorf("no sessionID received")
	}

	return SessionID(sessionID), nil
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
