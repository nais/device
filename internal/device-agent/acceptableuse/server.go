package acceptableuse

import (
	"html/template"
	"net/http"
	"time"
)

func NewServer(hasApproved bool, toggleApproval chan<- struct{}, keepAlive chan<- struct{}) *http.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("GET /", indexHandler(hasApproved))
	handler.HandleFunc("POST /toggle", func(http.ResponseWriter, *http.Request) {
		// TODO: provide some feedback to the user
		toggleApproval <- struct{}{}
	})
	handler.HandleFunc("GET /ping", func(http.ResponseWriter, *http.Request) {
		keepAlive <- struct{}{}
	})

	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func indexHandler(hasApproved bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		data := struct {
			HasApproved bool
		}{
			HasApproved: hasApproved,
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
