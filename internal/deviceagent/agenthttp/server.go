// Package agenthttp provides a simple HTTP server for serving psk-secured sites on localhost for the device-agent
package agenthttp

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/nais/device/internal/random"
	"golang.org/x/sync/errgroup"
)

var mux = &localMux{
	mux:    http.NewServeMux(),
	secret: random.RandomString(16, random.LettersAndNumbers),
}
var addr = ""

func init() {
	mux.mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "This server hosts the naisdevice local pages. It is not meant to be accessed directly, but rather through the naisdevice systray application or the Nais CLI.")
	})
}

type localMux struct {
	secret string
	mux    *http.ServeMux
}

func (m *localMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Received request for", req.URL.Path)

	// Disable auth for auth redirect uri's - maybe?
	if req.Method == http.MethodGet && (req.URL.Path == "/" || strings.HasPrefix(req.URL.Path, "/auth")) {
		m.mux.ServeHTTP(w, req)
		return
	}

	if req.URL.Query().Get("s") != m.secret {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	m.mux.ServeHTTP(w, req)
}

func Serve(listeners ...net.Listener) error {
	if len(listeners) == 0 {
		return errors.New("no listeners configured")
	}

	addr = listeners[0].Addr().String()
	addr = strings.Replace(addr, "127.0.0.1", "localhost", 1)
	addr = strings.Replace(addr, "[::1]", "localhost", 1)

	server := &http.Server{Handler: mux}
	eg := errgroup.Group{}
	closeOnce := sync.Once{}

	for _, listener := range listeners {
		eg.Go(func() error {
			err := server.Serve(listener)
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			if err != nil {
				closeOnce.Do(func() {
					_ = server.Close()
				})
			}
			return err
		})
	}

	return eg.Wait()
}

func Path(path string, withSecret bool) string {
	url := fmt.Sprintf("http://%s%s", addr, path)
	if withSecret {
		sep := "?"
		if strings.Contains(path, sep) {
			sep = "&"
		}

		url += sep + "s=" + mux.secret
	}
	return url
}

func HandleFunc(pattern string, handler http.HandlerFunc) {
	mux.mux.HandleFunc(pattern, handler)
}
