package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nais/device/pkg/apiserver/auth"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nais/device/pkg/device-agent/open"
	"github.com/nais/device/pkg/pb"

	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/apiserver/kekw"
)

func EnsureAuth(existing *pb.Session, ctx context.Context, apiserverURL, platform, serial string) (*pb.Session, error) {
	var err error

	if !existing.Expired() {
		return existing, nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("creating listener: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	var authURL string
	authURL, err = getAuthURL(apiserverURL, ctx, port)

	if err != nil {
		return nil, fmt.Errorf("getting Azure auth URL from apiserver: %w", err)
	}

	sessionInfo, err := RunFlow(ctx, urlOpener(authURL), MakeSessionInfoGetter(apiserverURL, platform, serial, port), listener)

	if err != nil {
		return nil, fmt.Errorf("ensuring valid session key: %v", err)
	}

	return sessionInfo, nil
}

type SessionInfoGetter func(context.Context, string) (*pb.Session, error)
type UrlOpener func() error

func RunFlow(ctx context.Context, urlOpener UrlOpener, exchange SessionInfoGetter, listener net.Listener) (*pb.Session, error) {
	handler := http.NewServeMux()

	sessionInfo := make(chan *pb.Session, 1)
	// define a handler that will get the authorization code, call the login endpoint to get a new session, and close the HTTP server
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Catch if user has not approved terms
		if strings.HasPrefix(r.URL.Query().Get("error_description"), "AADSTS50105") {
			http.Redirect(w, r, "https://naisdevice-approval.nais.io/", http.StatusSeeOther)
			sessionInfo <- nil
			return
		}

		si, err := exchange(ctx, r.URL.RawQuery)
		if err != nil {
			err = fmt.Errorf("failed logging in: %v", err)
			failureResponse(w, err.Error())
			sessionInfo <- nil
			return
		}

		successfulResponse(w, "Successfully authenticated 👌 Close me pls")
		sessionInfo <- si
	})

	server := &http.Server{Handler: handler}
	/* TODO
	   consider waiting for this to become ready. In the case where Azure AD
	   redirects extremely fast the listener won't be ready. We saw this in
	   unit tests where we mocked AAD.
	*/

	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("serving: %v", err)
		} else {
			log.Debugf("done serving")
		}
	}()
	defer server.Close()

	err := urlOpener()
	if err != nil {
		log.Errorf("opening browser, err: %v", err)
		// Don't return, as this is not fatal (user can open browser manually)
	}

	var si *pb.Session
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

func urlOpener(url string) UrlOpener {
	return func() error {
		err := open.Open(url)

		if err != nil {
			fmt.Printf("If the browser didn't open, visit this url to sign in: %v\n", url)
		}

		return err
	}
}

func MakeSessionInfoGetter(apiserverURL, platform, serial string, port int) SessionInfoGetter {
	return func(ctx context.Context, queryParams string) (*pb.Session, error) {
		codeRequestURL := url.URL{
			Scheme:   "http",
			Host:     strings.Split(apiserverURL, "://")[1],
			Path:     "/login",
			RawQuery: queryParams,
		}

		codeRequest, _ := http.NewRequest(http.MethodGet, codeRequestURL.String(), nil)
		codeRequest.Header.Add("x-naisdevice-platform", platform)
		codeRequest.Header.Add("x-naisdevice-serial", serial)
		codeRequest.Header.Add("x-naisdevice-listen-port", strconv.Itoa(port))

		resp, err := http.DefaultClient.Do(codeRequest.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("sending auth code to apiserver login: %v", err)
		}

		si := &auth.LegacySessionInfo{}
		if err := json.NewDecoder(resp.Body).Decode(si); err != nil {
			return nil, fmt.Errorf("reading session info from response body: %v", err)
		}

		return si.ToProtobuf(), nil
	}
}

func getAuthURL(apiserverURL string, ctx context.Context, port int) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiserverURL+"/authurl", nil)
	if err != nil {
		return "", fmt.Errorf("creating request to get Azure auth URL: %v", err)
	}
	req.Header.Add("x-naisdevice-listen-port", strconv.Itoa(port))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("getting Azure auth URL from apiserver: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to get Azure auth URL from apiserver (%v), http status: %v", apiserverURL, resp.Status)
	}

	authURL, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read response body: %v", err)
	}

	if len(authURL) == 0 {
		return "", fmt.Errorf("server returned empty auth URL")
	}

	return string(authURL), nil
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
