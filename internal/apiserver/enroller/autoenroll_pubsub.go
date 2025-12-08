package enroller

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/enroll"
	"github.com/nais/device/pkg/pb"
)

type autoEnrollConfig struct {
	GatewayTopicName        string `json:"gateway_topic_name"`
	GatewaySubscriptionName string `json:"gateway_subscription_name"`
	DeviceTopicName         string `json:"device_topic_name"`
	DeviceSubscriptionName  string `json:"device_subscription_name"`
	ExternalIP              string `json:"external_ip"`
}

type autoEnroll struct {
	db                   database.Database
	peers                []*pb.Gateway
	apiServerGRPCAddress string
	log                  logrus.FieldLogger
	gatewayTopic         *pubsub.Topic
	gatewaySubscription  *pubsub.Subscription
	deviceTopic          *pubsub.Topic
	deviceSubscription   *pubsub.Subscription
	externalIP           string
}

func NewAutoEnroll(
	ctx context.Context,
	db database.Database,
	peers []*pb.Gateway,
	apiServerGRPCAddress string,
	log logrus.FieldLogger,
) (*autoEnroll, error) {
	projectID, err := enroll.GetGoogleMetadataString(ctx, "project/project-id", log)
	if err != nil {
		log.WithError(err).Fatal("failed to get project ID")
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	b, err := enroll.GetGoogleMetadata(ctx, "instance/attributes/auto-enroll-config", log)
	if err != nil {
		return nil, err
	}

	ec := &autoEnrollConfig{}
	if err := json.Unmarshal(b, ec); err != nil {
		return nil, err
	}

	return &autoEnroll{
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

func (a *autoEnroll) Run(ctx context.Context) error {
	a.log.WithFields(logrus.Fields{
		"topic": a.gatewayTopic.String(),
		"sub":   a.gatewaySubscription.String(),
	}).Info("starting auto enroll...")
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return a.gatewaySubscription.Receive(ctx, a.receiveGateway)
	})
	eg.Go(func() error {
		return a.deviceSubscription.Receive(ctx, a.receiveDevice)
	})

	return eg.Wait()
}

func (a *autoEnroll) receiveGateway(ctx context.Context, msg *pubsub.Message) {
	defer msg.Ack()

	if msg.Attributes["type"] != enroll.TypeEnrollRequest {
		a.log.WithField("attributes", msg.Attributes).Debug("ignoring pubsub message")
		msg.Nack()
		return
	}

	var req *enroll.GatewayRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		a.log.WithError(err).Error("failed to unmarshal request")
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
		log.WithError(err).Error("failed to add gateway")
		return
	}

	gw, err := a.db.ReadGateway(ctx, req.Name)
	if err != nil {
		msg.Nack()
		log.WithError(err).Error("failed to get gateway")
		return
	}

	resp := enroll.Response{
		APIServerGRPCAddress: a.apiServerGRPCAddress,
		WireGuardIPv4:        gw.Ipv4 + "/21",
		WireGuardIPv6:        gw.Ipv6 + "/64",
		Peers:                a.peers,
	}

	b, err := json.Marshal(&resp)
	if err != nil {
		msg.Nack()
		log.WithError(err).Error("failed to marshal response")
		return
	}

	pubresp := a.gatewayTopic.Publish(ctx, &pubsub.Message{
		Data: b,
		Attributes: map[string]string{
			"type":   enroll.TypeEnrollResponse,
			"source": "apiserver",
			"target": req.Name,
		},
	})
	_, err = pubresp.Get(ctx)
	if err != nil {
		log.WithError(err).Error("failed to publish response")
	}

	log.Info("enrolled gateway")
}

func (a *autoEnroll) receiveDevice(ctx context.Context, msg *pubsub.Message) {
	defer msg.Ack()

	if msg.Attributes["type"] != enroll.TypeEnrollRequest {
		a.log.WithField("attributes", msg.Attributes).Debug("ignoring pubsub message")
		msg.Nack()
		return
	}

	var req *enroll.DeviceRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		a.log.WithError(err).Error("failed to unmarshal request")
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
		log.WithError(err).Error("failed to add device")
		return
	}

	gw, err := a.db.ReadDeviceBySerialPlatform(ctx, req.Serial, req.Platform)
	if err != nil {
		msg.Nack()
		log.WithError(err).Error("failed to get device")
		return
	}

	resp := enroll.Response{
		APIServerGRPCAddress: a.apiServerGRPCAddress,
		WireGuardIPv4:        gw.Ipv4,
		WireGuardIPv6:        gw.Ipv6,
		Peers:                a.peers,
	}

	b, err := json.Marshal(&resp)
	if err != nil {
		msg.Nack()
		log.WithError(err).Error("failed to marshal response")
		return
	}

	pubresp := a.deviceTopic.Publish(ctx, &pubsub.Message{
		Data: b,
		Attributes: map[string]string{
			"type":    enroll.TypeEnrollResponse,
			"source":  "apiserver",
			"subject": msg.Attributes["subject"],
		},
	})
	_, err = pubresp.Get(ctx)
	if err != nil {
		log.WithError(err).Error("failed to publish response")
	}

	log.Info("enrolled device")
}
