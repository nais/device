package enroller

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/pubsubenroll"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type autoEnrollConfig struct {
	GatewayTopicName        string `json:"gateway_topic_name"`
	GatewaySubscriptionName string `json:"gateway_subscription_name"`
	DeviceTopicName         string `json:"device_topic_name"`
	DeviceSubscriptionName  string `json:"device_subscription_name"`
	ExternalIP              string `json:"external_ip"`
}

type AutoEnroll struct {
	db                   database.APIServer
	peers                []*pb.Gateway
	apiServerGRPCAddress string
	log                  *logrus.Entry
	gatewayTopic         *pubsub.Topic
	gatewaySubscription  *pubsub.Subscription
	deviceTopic          *pubsub.Topic
	deviceSubscription   *pubsub.Subscription
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
		gatewayTopic:         client.Topic(ec.GatewayTopicName),
		gatewaySubscription:  client.Subscription(ec.GatewaySubscriptionName),
		deviceTopic:          client.Topic(ec.DeviceTopicName),
		deviceSubscription:   client.Subscription(ec.DeviceSubscriptionName),
		externalIP:           ec.ExternalIP,
		log:                  log,
	}, nil
}

func (a *AutoEnroll) Run(ctx context.Context) error {
	a.log.WithFields(logrus.Fields{
		"topic": a.gatewayTopic.String(),
		"sub":   a.gatewaySubscription.String(),
	}).Info("Starting auto enroll...")
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return a.gatewaySubscription.Receive(ctx, a.receiveGateway)
	})
	eg.Go(func() error {
		return a.deviceSubscription.Receive(ctx, a.receiveDevice)
	})

	return eg.Wait()
}

func (a *AutoEnroll) receiveGateway(ctx context.Context, msg *pubsub.Message) {
	defer msg.Ack()

	if msg.Attributes["type"] != pubsubenroll.TypeEnrollRequest {
		a.log.Debugf("ignoring pubsub message with attribtes: %#v", msg.Attributes)
		msg.Nack()
		return
	}

	var req *pubsubenroll.GatewayRequest
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

	pubresp := a.gatewayTopic.Publish(ctx, &pubsub.Message{
		Data: b,
		Attributes: map[string]string{
			"type":   pubsubenroll.TypeEnrollResponse,
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

func (a *AutoEnroll) receiveDevice(ctx context.Context, msg *pubsub.Message) {
	defer msg.Ack()

	if msg.Attributes["type"] != pubsubenroll.TypeEnrollRequest {
		a.log.Debugf("ignoring pubsub message with attribtes: %#v", msg.Attributes)
		msg.Nack()
		return
	}

	var req *pubsubenroll.DeviceRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		a.log.WithError(err).Error("Failed to unmarshal request")
		return
	}
	log := a.log.WithFields(logrus.Fields{"serial": req.Serial, "platform": req.Platform})

	err := a.db.AddDevice(ctx, &pb.Device{
		Username:  req.Owner,
		PublicKey: string(req.WireGuardPublicKey),
		Serial:    req.Serial,
		Platform:  req.Platform,
	})
	if err != nil {
		msg.Nack()
		log.WithError(err).Error("Failed to add device")
		return
	}

	gw, err := a.db.ReadDeviceBySerialPlatform(ctx, req.Serial, req.Platform)
	if err != nil {
		msg.Nack()
		log.WithError(err).Error("Failed to get device")
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

	pubresp := a.deviceTopic.Publish(ctx, &pubsub.Message{
		Data: b,
		Attributes: map[string]string{
			"type":    pubsubenroll.TypeEnrollResponse,
			"source":  "apiserver",
			"subject": msg.Attributes["subject"],
		},
	})
	_, err = pubresp.Get(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to publish response")
	}

	log.Infof("Enrolled device")
}
