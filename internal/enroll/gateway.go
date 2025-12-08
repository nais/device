package enroll

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"cloud.google.com/go/pubsub"
	"github.com/nais/device/internal/ioconvenience"
	"github.com/sirupsen/logrus"
)

type GatewayClient struct {
	wireGuardPublicKey []byte
	port               int
	hashedPassword     string
	log                logrus.FieldLogger

	Name             string `json:"name"`
	EnrollProjectID  string `json:"project_id"`
	TopicName        string `json:"topic_name"`
	SubscriptionName string `json:"subscription_name"`
	ExternalIP       string `json:"external_ip"`
}

func NewGatewayClient(ctx context.Context, publicKey []byte, hashedPassword string, wireguardListenPort int, log logrus.FieldLogger) (*GatewayClient, error) {
	b, err := GetGoogleMetadata(ctx, "instance/attributes/enroll-config", log)
	if err != nil {
		return nil, err
	}

	ec := &GatewayClient{
		port:               wireguardListenPort,
		wireGuardPublicKey: publicKey,
		hashedPassword:     hashedPassword,
		log:                log,
	}
	if err := json.Unmarshal(b, ec); err != nil {
		return nil, err
	}
	return ec, nil
}

func (c *GatewayClient) Bootstrap(ctx context.Context) (*Response, error) {
	c.log.Info("bootstrapping...")
	projectID, err := GetGoogleMetadataString(ctx, "project/project-id", c.log)
	if err != nil {
		return nil, fmt.Errorf("get google metadata: %w", err)
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create pubsub client: %w", err)
	}

	topic := client.TopicInProject(c.TopicName, c.EnrollProjectID)
	sub := client.Subscription(c.SubscriptionName)

	c.log.WithFields(logrus.Fields{"topic": topic.String(), "subscription": sub.String()}).Info("bootstrap using pubsub")

	if err := c.publishAndWait(ctx, topic); err != nil {
		return nil, fmt.Errorf("publish and wait: %w", err)
	}

	var resp *Response
	var unmarshalErr error
	ctx, cancel := context.WithCancel(ctx)
	err = sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
		if v, ok := msg.Attributes["type"]; !ok || v != TypeEnrollResponse {
			msg.Nack()
			return
		}

		c.log.Debug("received enroll-response")
		msg.Ack()
		resp = &Response{}
		unmarshalErr = json.Unmarshal(msg.Data, resp)
		cancel()
	})

	c.log.WithError(err).WithFields(logrus.Fields{
		"can":      errors.Is(err, context.Canceled),
		"deadline": errors.Is(err, context.DeadlineExceeded),
	}).Debug("receive err")
	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, fmt.Errorf("bootstrap failed: %w", err)
	}

	err = ctx.Err()
	c.log.WithError(err).WithFields(logrus.Fields{
		"can":      errors.Is(err, context.Canceled),
		"deadline": errors.Is(err, context.DeadlineExceeded),
	}).Debug("ctx err")
	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, fmt.Errorf("bootstrap failed: %w", err)
	}

	if unmarshalErr != nil {
		return nil, fmt.Errorf("parse json: %w", unmarshalErr)
	}

	if resp == nil {
		return nil, fmt.Errorf("no resp")
	}

	return resp, nil
}

func (c *GatewayClient) publishAndWait(ctx context.Context, topic *pubsub.Topic) error {
	enrollMsg := GatewayRequest{
		WireGuardPublicKey: c.wireGuardPublicKey,
		Name:               c.Name,
		Endpoint:           net.JoinHostPort(c.ExternalIP, strconv.Itoa(c.port)),
		HashedPassword:     c.hashedPassword,
	}
	b, err := json.Marshal(enrollMsg)
	if err != nil {
		return err
	}
	pubres := topic.Publish(ctx, &pubsub.Message{
		Data: b,
		Attributes: map[string]string{
			"source": "gateway-agent",
			"type":   TypeEnrollRequest,
			"name":   c.Name,
		},
	})

	<-pubres.Ready()
	return nil
}

func GetGoogleMetadataString(ctx context.Context, path string, log logrus.FieldLogger) (string, error) {
	b, err := GetGoogleMetadata(ctx, path, log)
	return string(b), err
}

func GetGoogleMetadata(ctx context.Context, path string, log logrus.FieldLogger) ([]byte, error) {
	url := "http://metadata.google.internal/computeMetadata/v1/" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer ioconvenience.CloseWithLog(log, resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status on metadata request: %v", resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}
