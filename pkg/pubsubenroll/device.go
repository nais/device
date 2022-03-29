package pubsubenroll

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

type DeviceRequest struct {
	Platform           string `json:"platform"`
	Owner              string `json:"owner"`
	Serial             string `json:"serial"`
	WireGuardPublicKey []byte `json:"wireguard_public_key"`
}

type responseOrError struct {
	Response *Response
	Error    error
}

func Enroll(ctx context.Context, req *DeviceRequest, token *oauth2.Token, projectID, topicName string, log *logrus.Entry) (*Response, error) {
	subscriptionUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	creds := &google.Credentials{TokenSource: oauth2.StaticTokenSource(token)}
	client, err := pubsub.NewClient(ctx, projectID, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}

	topic := client.Topic(topicName)
	subscription, err := client.CreateSubscription(ctx, subscriptionUUID.String(), pubsub.SubscriptionConfig{
		Topic:             topic,
		RetentionDuration: 10 * time.Minute,
		ExpirationPolicy:  24 * time.Hour,
		Labels: map[string]string{
			"serial":   req.Serial,
			"owner":    req.Owner,
			"platform": req.Platform,
		},
		Filter: fmt.Sprintf("target = \"%s\"", subscriptionUUID.String()),
	})
	if err != nil {
		return nil, err
	}
	defer subscription.Delete(context.Background())

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err = publish(ctx, req, topic, subscriptionUUID.String(), log)
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

		resp := &Response{}
		err := json.Unmarshal(msg.Data, resp)
		if err != nil {
			responseChan <- &responseOrError{Error: err}
			return
		}

		responseChan <- &responseOrError{Response: resp}
	}
}

func publish(ctx context.Context, req *DeviceRequest, topic *pubsub.Topic, subscription string, log *logrus.Entry) error {
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}

	msg := &pubsub.Message{
		Data: b,
		Attributes: map[string]string{
			"source":       "device-agent",
			"type":         TypeEnrollRequest,
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
