package enroller

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/pubsubenroll"
	"github.com/sirupsen/logrus"
)

type autoEnrollConfig struct {
	TopicName        string `json:"topic_name"`
	ExternalIP       string `json:"external_ip"`
	SubscriptionName string `json:"subscription_name"`
}

type AutoEnroll struct {
	db                   database.APIServer
	peers                []*pb.Gateway
	apiServerGRPCAddress string
	log                  *logrus.Entry
	topic                *pubsub.Topic
	subscription         *pubsub.Subscription
	externalIP           string
}

func NewAutoEnroll(
	ctx context.Context,
	db database.APIServer,
	peers []*pb.Gateway,
	apiServerGRPCAddress string,
	log *logrus.Entry,
) (*AutoEnroll, error) {
	projectID, err := pubsubenroll.GetGoogleMetadataString(ctx, "project/project-id")
	if err != nil {
		log.WithError(err).Fatal("Failed to get project ID")
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	b, err := pubsubenroll.GetGoogleMetadata(ctx, "instance/attributes/auto-enroll-config")
	if err != nil {
		return nil, err
	}

	ec := &autoEnrollConfig{}
	if err := json.Unmarshal(b, ec); err != nil {
		return nil, err
	}

	return &AutoEnroll{
		db:                   db,
		peers:                peers,
		apiServerGRPCAddress: apiServerGRPCAddress,
		topic:                client.Topic(ec.TopicName),
		subscription:         client.Subscription(ec.SubscriptionName),
		externalIP:           ec.ExternalIP,
		log:                  log,
	}, nil
}

func (a *AutoEnroll) Run(ctx context.Context) error {
	a.log.Infof("Starting auto enroll...")
	return a.subscription.Receive(ctx, a.receive)
}

func (a *AutoEnroll) receive(ctx context.Context, msg *pubsub.Message) {
	defer msg.Ack()

	if msg.Attributes["type"] != "enroll-request" {
		msg.Nack()
		return
	}

	var req *pubsubenroll.Request
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		a.log.WithError(err).Error("Failed to unmarshal request")
		return
	}
	log := a.log.WithField("gateway", req.Name)

	err := a.db.AddGateway(ctx, &pb.Gateway{
		Name:         req.Name,
		PublicKey:    string(req.WireGuardPublicKey),
		Endpoint:     req.Endpoint,
		PasswordHash: req.HashedPassword,
	})
	if err != nil {
		msg.Nack()
		log.WithError(err).Error("Failed to add gateway")
		return
	}

	gw, err := a.db.ReadGateway(ctx, req.Name)
	if err != nil {
		msg.Nack()
		log.WithError(err).Error("Failed to get gateway")
		return
	}

	resp := pubsubenroll.Response{
		APIServerGRPCAddress: a.apiServerGRPCAddress,
		WireGuardIP:          gw.Ip,
		Peers:                a.peers,
	}

	b, err := json.Marshal(&resp)
	if err != nil {
		msg.Nack()
		log.WithError(err).Error("Failed to marshal response")
		return
	}

	pubresp := a.topic.Publish(ctx, &pubsub.Message{
		Data: b,
		Attributes: map[string]string{
			"type":   "enroll-response",
			"source": "apiserver",
			"target": req.Name,
		},
	})
	_, err = pubresp.Get(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to publish response")
	}

	log.Infof("Enrolled gateway")
}
