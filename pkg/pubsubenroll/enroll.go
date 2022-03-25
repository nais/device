package pubsubenroll

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
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

type Request struct {
	WireGuardPublicKey []byte `json:"wireguard_public_key"`
	Name               string `json:"name"`
	Endpoint           string `json:"endpoint"`
	HashedPassword     string `json:"hashed_password"`
}

type Response struct {
	APIServerGRPCAddress string        `json:"api_server_grpc_address"`
	WireGuardIP          string        `json:"wireguard_ip"`
	Peers                []*pb.Gateway `json:"peers"`
}

type Client struct {
	wireGuardPublicKey []byte
	port               int
	hashedPassword     string

	Name             string `json:"name"`
	EnrollProjectID  string `json:"project_id"`
	TopicName        string `json:"topic_name"`
	SubscriptionName string `json:"subscription_name"`
	ExternalIP       string `json:"external_ip"`
}

func New(ctx context.Context, publicKey []byte, hashedPassword string, wireguardListenPort int, log *log.Entry) (*Client, error) {
	b, err := GetGoogleMetadata(ctx, "instance/attributes/enroll-config")
	if err != nil {
		return nil, err
	}

	ec := &Client{
		port:               wireguardListenPort,
		wireGuardPublicKey: publicKey,
		hashedPassword:     hashedPassword,
	}
	if err := json.Unmarshal(b, ec); err != nil {
		return nil, err
	}
	return ec, nil
}

func (c *Client) Bootstrap(ctx context.Context) (*Response, error) {
	log.Info("Bootstrapping...")
	projectID, err := GetGoogleMetadataString(ctx, "project/project-id")
	if err != nil {
		return nil, err
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	topic := client.TopicInProject(c.TopicName, c.EnrollProjectID)
	sub := client.Subscription(c.SubscriptionName)

	if err := c.publishAndWait(ctx, topic); err != nil {
		return nil, err
	}

	var resp *Response
	ctx, cancel := context.WithCancel(ctx)
	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		if v, ok := msg.Attributes["type"]; !ok || v != "enroll-response" {
			msg.Nack()
			return
		}

		msg.Ack()
		if err := json.Unmarshal(msg.Data, resp); err != nil {
			log.WithError(err).Error("unable to parse enroll response")
		}
		cancel()
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, err
	}

	return resp, nil
}

func (c *Client) publishAndWait(ctx context.Context, topic *pubsub.Topic) error {
	enrollMsg := Request{
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
			"type":   "enroll-request",
			"name":   c.Name,
		},
	})

	<-pubres.Ready()
	return nil
}

func GetGoogleMetadataString(ctx context.Context, path string) (string, error) {
	b, err := GetGoogleMetadata(ctx, path)
	return string(b), err
}

func GetGoogleMetadata(ctx context.Context, path string) ([]byte, error) {
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
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 status on metadata request: %v", resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return b, nil
}
