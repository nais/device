package pubsubenroll

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/sirupsen/logrus"
)

type Worker interface {
	Run(ctx context.Context) error
	Send(ctx context.Context, req *DeviceRequest) (*Response, error)
}

type worker struct {
	log          *logrus.Entry
	topic        *pubsub.Topic
	subscription *pubsub.Subscription

	lock  sync.Mutex
	queue map[string]chan *Response
}

func NewWorker(ctx context.Context, log *logrus.Entry) (Worker, error) {
	projectID := os.Getenv("GCP_PROJECT")
	topicName := os.Getenv("PUBSUB_TOPIC")
	subscriptionName := os.Getenv("PUBSUB_SUBSCRIPTION")
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &worker{
		log:          log,
		topic:        client.Topic(topicName),
		subscription: client.Subscription(subscriptionName),
		queue:        make(map[string]chan *Response),
	}, nil
}

func (w *worker) Run(ctx context.Context) error {
	return w.subscription.Receive(ctx, w.receive)
}

func (w *worker) receive(ctx context.Context, msg *pubsub.Message) {
	defer msg.Ack()

	subject := msg.Attributes["subject"]
	if subject == "" {
		w.log.WithField("attributes", msg.Attributes).Error("missing subject")
		msg.Nack()
		return
	}

	var resp Response
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

func (w *worker) Send(ctx context.Context, req *DeviceRequest) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	id := req.Serial + "-" + req.Platform
	attrs := map[string]string{
		"source":  "enroller",
		"type":    TypeEnrollRequest,
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

	ch := make(chan *Response, 1)
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
