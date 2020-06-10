package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nais/device/apiserver/kekw"
	log "github.com/sirupsen/logrus"
)

type SessionInfo struct {
	Key    string `json:"key"`
	Expiry int64  `json:"expiry"`
}

func RunFlow(ctx context.Context, authURL, apiserverURL, platform, serial string) (*SessionInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	handler := http.NewServeMux()

	sessionInfo := make(chan *SessionInfo)
	// define a handler that will get the authorization code, call the login endpoint to get a new session, and close the HTTP server
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Host = apiserverURL
		r.Header.Add("x-naisdevice-platform", platform)
		r.Header.Add("x-naisdevice-serial", serial)

		resp, err := http.DefaultClient.Do(r.WithContext(ctx))
		if err != nil {
			log.Errorf("Sending auth code to apiserver login: %v", err)
			sessionInfo <- nil
			return
		}

		var si SessionInfo
		if err := json.NewDecoder(resp.Body).Decode(&si); err != nil {
			log.Errorf("Reading session info from response body: %v", err)
			sessionInfo <- nil
			return
		}

		successfulResponse(w, "Successfully authenticated 👌 Close me pls")
		sessionInfo <- &si
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

	var si *SessionInfo
	select {
	case si = <-sessionInfo:
		break
	case <-time.After(3 * time.Minute):
		log.Warn("timed out waiting for authentication flow")
		break
	}

	if si == nil {
		return nil, fmt.Errorf("no session info received")
	}

	return si, nil
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
