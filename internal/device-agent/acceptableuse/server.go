package acceptableuse

import (
	"html/template"
	"net/http"
	"strconv"
	"time"
)

func auth(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("s") != secret {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next(w, req)
	}
}

func NewServer(secret string, acceptedAt time.Time, setAccepted func(accepted bool) error, keepAlive chan<- struct{}) *http.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("GET /", auth(secret, indexHandler(acceptedAt)))
	handler.HandleFunc("POST /setAccepted", auth(secret, func(w http.ResponseWriter, req *http.Request) {
		accepted, err := strconv.ParseBool(req.FormValue("accepted"))
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if err := setAccepted(accepted); err != nil {
			http.Error(w, "Failed to set acceptance status.", http.StatusInternalServerError)
			return
		}

		_, _ = w.Write([]byte("You may now close this window."))
	}))
	handler.HandleFunc("GET /ping", auth(secret, func(http.ResponseWriter, *http.Request) {
		keepAlive <- struct{}{}
	}))

	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func indexHandler(acceptedAt time.Time) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		data := struct {
			HasAccepted bool
			AcceptedAt  string
		}{
			HasAccepted: !acceptedAt.IsZero(),
			AcceptedAt:  acceptedAt.Local().Format(time.RFC822),
		}

		t, err := template.ParseFS(templates, "templates/index.html")
		if err != nil {
			http.Error(w, "Failed to parse templates.", http.StatusInternalServerError)
			return
		}

		if err := t.ExecuteTemplate(w, "index.html", data); err != nil {
			http.Error(w, "Failed to render index page.", http.StatusInternalServerError)
			return
		}
	}
}
