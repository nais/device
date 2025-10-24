package enroll

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
)

type localWorker struct {
	log        logrus.FieldLogger
	listenAddr string
	lock       sync.Mutex

	data     []json.RawMessage
	response chan *Response
}

func NewLocal(ctx context.Context, listenAddr string, log logrus.FieldLogger) (Worker, error) {
	return &localWorker{
		log:        log,
		listenAddr: listenAddr,
	}, nil
}

func (w *localWorker) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/enrollments", func(wr http.ResponseWriter, r *http.Request) {
		w.lock.Lock()
		defer w.lock.Unlock()
		switch r.Method {
		case http.MethodGet:
			all, err := json.Marshal(w.data)
			if err != nil {
				http.Error(wr, err.Error(), http.StatusInternalServerError)
				return
			}
			wr.Header().Set("Content-Type", "application/json")
			_, _ = wr.Write(all)
			w.data = nil
		case http.MethodPost:
			if w.response == nil {
				http.Error(wr, "no response channel", http.StatusInternalServerError)
				w.log.Error("no response channel")
				return
			}

			enrollResponse := &Response{}
			err := json.NewDecoder(r.Body).Decode(enrollResponse)
			if err != nil {
				w.log.WithError(err).Errorf("received enrollment response: %v", enrollResponse)
				http.Error(wr, "decode json", http.StatusInternalServerError)
				return
			}
			w.log.Infof("received enrollment response: %v", enrollResponse)
			w.response <- enrollResponse
		}
	})

	server := &http.Server{Addr: w.listenAddr, Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			w.log.WithError(err).Error("http server")
		}
	}()

	<-ctx.Done()
	_ = server.Close()

	return ctx.Err()
}

func (w *localWorker) Send(ctx context.Context, req *DeviceRequest) (*Response, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	w.lock.Lock()
	w.data = append(w.data, b)
	w.response = make(chan *Response)
	w.lock.Unlock()

	return <-w.response, nil
}
