// Package agenthttp provides a simple HTTP server for serving psk-secured sites on localhost for the device-agent
package agenthttp

import (
	"fmt"
	"net"
	"net/http"

	"github.com/nais/device/internal/random"
)

var mux = &localMux{
	mux:    http.NewServeMux(),
	secret: random.RandomString(32, random.LettersAndNumbers),
}
var addr = ""

func init() {
	mux.mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, "This server hosts the naisdevice local pages. It is not meant to be accessed directly, but rather through the naisdevice systray application or the Nais CLI.")
	})
}

type localMux struct {
	secret string
	mux    *http.ServeMux
}

func (m *localMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Received request for", req.URL.Path)
	if req.Method == http.MethodGet && (req.URL.Path == "/" || req.URL.Path == "/google") {
		fmt.Println("Serving unprotected path", req.URL.Path)
		m.mux.ServeHTTP(w, req)
		return
	}

	if req.URL.Query().Get("s") != m.secret {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	m.mux.ServeHTTP(w, req)
}

func Serve(listener net.Listener) error {
	server := &http.Server{Handler: mux}
	addr = listener.Addr().String()
	return server.Serve(listener)
}

func Secret() string {
	return mux.secret
}

func Addr() string {
	return addr
}

func HandleFunc(pattern string, handler http.HandlerFunc) {
	mux.mux.HandleFunc(pattern, handler)
}
