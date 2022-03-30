package enroll

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/google/uuid"
	"github.com/nais/device/pkg/pubsubenroll"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

func init() {
	functions.HTTP("enroll", enrollHandler)
}

type responseOrError struct {
	Response *pubsubenroll.Response
	Error    error
}

func enrollHandler(w http.ResponseWriter, r *http.Request) {
	var req pubsubenroll.DeviceRequest
	projectID := os.Getenv("GCP_PROJECT")
	topicName := os.Getenv("PUBSUB_TOPIC")

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := enroll(r.Context(), &req, nil, projectID, topicName, logrus.WithField("component", "enroll"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func enroll(ctx context.Context, req *pubsubenroll.DeviceRequest, token *oauth2.Token, projectID, topicName string, log *logrus.Entry) (*pubsubenroll.Response, error) {
	subscriptionUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	subscriptionName := "enroll-" + subscriptionUUID.String()

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	topic := client.Topic(topicName)
	subscription, err := client.CreateSubscription(ctx, subscriptionName, pubsub.SubscriptionConfig{
		Topic:             topic,
		RetentionDuration: 10 * time.Minute,
		ExpirationPolicy:  24 * time.Hour,
		Labels: map[string]string{
			"serial":   strings.ToLower(req.Serial),
			"platform": strings.ToLower(req.Platform),
		},
		Filter: fmt.Sprintf("attributes.target = \"%s\"", subscriptionName),
	})
	if err != nil {
		return nil, err
	}
	defer subscription.Delete(context.Background())

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err = publish(ctx, req, topic, subscriptionName, log)
	if err != nil {
		return nil, err
	}

	responseChan := make(chan *responseOrError, 1)
	go func() {
		err := subscription.Receive(ctx, receive(responseChan, log))
		if err != nil {
			log.WithError(err).Error("receive")
			cancel()
		}
	}()

	select {
	case responseOrError := <-responseChan:
		return responseOrError.Response, responseOrError.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func receive(responseChan chan *responseOrError, log *logrus.Entry) func(ctx context.Context, msg *pubsub.Message) {
	return func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack()

		log.WithFields(logrus.Fields{
			"attributes": msg.Attributes,
		}).Debug("receive message")

		resp := &pubsubenroll.Response{}
		err := json.Unmarshal(msg.Data, resp)
		if err != nil {
			responseChan <- &responseOrError{Error: err}
			return
		}

		responseChan <- &responseOrError{Response: resp}
	}
}

func publish(ctx context.Context, req *pubsubenroll.DeviceRequest, topic *pubsub.Topic, subscription string, log *logrus.Entry) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	msg := &pubsub.Message{
		Data: b,
		Attributes: map[string]string{
			"source":       "device-agent",
			"type":         pubsubenroll.TypeEnrollRequest,
			"subscription": subscription,
		},
	}

	log.WithFields(logrus.Fields{
		"attributes": msg.Attributes,
		"req":        req,
	}).Debug("publish message")

	pubres := topic.Publish(ctx, msg)

	<-pubres.Ready()
	return nil
}
