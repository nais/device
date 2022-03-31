package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/nais/device/pkg/pubsubenroll"
	"github.com/sirupsen/logrus"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	log := logrus.New()
	log.Formatter = &logrus.JSONFormatter{}

	worker, err := newWorker(context.Background(), log.WithField("component", "worker"))
	if err != nil {
		log.WithError(err).Fatal("new worker")
		return
	}

	server := http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 3 * time.Second,
		IdleTimeout:       10 * time.Minute,
	}

	h := &Handler{
		log:    log.WithField("component", "enroller"),
		worker: worker,
	}

	http.Handle("/enroll", h)

	log.WithField("addr", ":"+port).Info("starting server")
	ctx, cancel := context.WithCancel(ctx)
	go logErr(log, cancel, func() error { return worker.Run(ctx) })
	go logErr(log, cancel, server.ListenAndServe)

	<-ctx.Done()

	// Reset os.Interrupt default behavior, similar to signal.Reset
	stop()
	log.Info("shutting down gracefully, press Ctrl+C again to force")

	// Gievn 5s more to process existing requests
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(timeoutCtx); err != nil {
		log.Error(err)
	}
}

type Worker struct {
	log          *logrus.Entry
	topic        *pubsub.Topic
	subscription *pubsub.Subscription

	lock  sync.Mutex
	queue map[string]chan *pubsubenroll.Response
}

func newWorker(ctx context.Context, log *logrus.Entry) (*Worker, error) {
	projectID := os.Getenv("GCP_PROJECT")
	topicName := os.Getenv("PUBSUB_TOPIC")
	subscriptionName := os.Getenv("PUBSUB_SUBSCRIPTION")
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &Worker{
		log:          log,
		topic:        client.Topic(topicName),
		subscription: client.Subscription(subscriptionName),
		queue:        make(map[string]chan *pubsubenroll.Response),
	}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	return w.subscription.Receive(ctx, w.receive)
}

func (w *Worker) receive(ctx context.Context, msg *pubsub.Message) {
	defer msg.Ack()

	subject := msg.Attributes["subject"]
	if subject == "" {
		w.log.WithFields(logrus.Fields{
			"attributes": msg.Attributes,
		}).Error("missing subject")
		msg.Nack()
		return
	}

	var resp pubsubenroll.Response
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		w.log.WithError(err).Error("unmarshal")
		return
	}

	w.lock.Lock()
	defer w.lock.Unlock()

	if ch, ok := w.queue[subject]; ok {
		ch <- &resp
		delete(w.queue, subject)
	} else {
		msg.Nack()
	}
}

func (w *Worker) Send(ctx context.Context, req *pubsubenroll.DeviceRequest) (*pubsubenroll.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	id := req.Serial + "-" + req.Platform
	attrs := map[string]string{
		"source":  "enroller",
		"type":    pubsubenroll.TypeEnrollRequest,
		"subject": id,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	msg := &pubsub.Message{
		Data:       b,
		Attributes: attrs,
	}

	if _, err := w.topic.Publish(ctx, msg).Get(ctx); err != nil {
		return nil, err
	}

	ch := make(chan *pubsubenroll.Response, 1)
	w.lock.Lock()
	w.queue[id] = ch
	w.lock.Unlock()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout")
	case resp := <-ch:
		return resp, nil
	}
}

type Handler struct {
	log    *logrus.Entry
	worker *Worker
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req pubsubenroll.DeviceRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := h.worker.Send(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func logErr(log *logrus.Logger, cancel context.CancelFunc, fn func() error) {
	if err := fn(); err != nil {
		cancel()
		log.WithError(err).Error("error")
	}
}
