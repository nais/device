package kolide

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/nais/device/internal/ioconvenience"
	kolidepb "github.com/nais/kolide-event-handler/pkg/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientInterceptor struct {
	RequireTLS bool
	Token      string
}

func (c *ClientInterceptor) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": c.Token,
	}, nil
}

func (c *ClientInterceptor) RequireTransportSecurity() bool {
	return c.RequireTLS
}

const (
	eventStreamBackoff = 10 * time.Second
)

func DeviceEventStreamer(ctx context.Context, log logrus.FieldLogger, grpcAddress, grpcToken string, grpcSecure bool, stream chan<- *kolidepb.DeviceEvent) error {
	interceptor := &ClientInterceptor{
		RequireTLS: grpcSecure,
		Token:      grpcToken,
	}

	dialOpts := make([]grpc.DialOption, 0)

	if grpcSecure {
		cred := credentials.NewTLS(&tls.Config{})
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(cred))
		dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(interceptor))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(
		grpcAddress,
		dialOpts...,
	)
	if err != nil {
		return err
	}

	defer ioconvenience.CloseWithLog(log, conn)

	s := kolidepb.NewKolideEventHandlerClient(conn)

	for ctx.Err() == nil {
		events, err := s.Events(ctx, &kolidepb.EventsRequest{})
		if err != nil {
			log.WithError(err).Error("start Kolide event stream")
			log.WithField("backoff", eventStreamBackoff).Warn("restarting event stream after backoff...")
			time.Sleep(eventStreamBackoff)
			continue
		}

		log.WithField("address", conn.Target()).Info("started Kolide event stream")

		for {
			event, err := events.Recv()
			if err != nil {
				log.WithError(err).Error("receive Kolide event")
				log.WithField("backoff", eventStreamBackoff).Warn("restarting event stream after backoff...")
				time.Sleep(eventStreamBackoff)
				break
			}

			stream <- event
		}
	}

	return ctx.Err()
}
