package kolide

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/nais/device/apiserver/database"
	kolideclient "github.com/nais/device/pkg/kolide-client"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/nais/kolide-event-handler/pkg/pb"
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

type Handler struct {
	kolideClient *kolideclient.KolideClient
	grpcToken    string
	grpcAddress  string
	checkDevices chan <- []*kolideclient.Device
}

func New(kolideApiToken, grpcToken, grpcAddress string) *Handler {
	return &Handler{
		kolideClient: kolideclient.New(kolideApiToken),
		grpcToken:    grpcToken,
		grpcAddress:  grpcAddress,
		checkDevices: make(chan []*kolideclient.Device, 50),
	}
}

func (handler *Handler) DeviceEventHandler(ctx context.Context) {
	interceptor := &ClientInterceptor{
		RequireTLS: false,
		Token:      handler.grpcToken,
	}

	cred := credentials.NewTLS(&tls.Config{})
	conn, err := grpc.DialContext(ctx, handler.grpcAddress, grpc.WithTransportCredentials(cred), grpc.WithPerRPCCredentials(interceptor))
	if err != nil {
		log.Errorf("connecting to grpc server: %v", err)
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Errorf("closing grpc connection: %v", err)
		}
	}()

	s := pb.NewKolideEventHandlerClient(conn)

	for {
		events, err := s.Events(ctx, &pb.EventsRequest{})
		if err != nil {
			if status.Code(err) == codes.Canceled {
				log.Infof("program finished")
				break
			}

			log.Errorf("calling rpc: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Infof("connected to %v", conn.Target())

		for {
			event, err := events.Recv()
			if err != nil {
				log.Warningf("receiving event: %v", err)
				time.Sleep(1 * time.Second)
				break
			}

			log.Infof("event received: %+v", event)

			device, err := handler.kolideClient.GetDevice(ctx, event.DeviceId)
			if err != nil {
				log.Warningf("get device: %v", err)
				continue
			}

			handler.checkDevices <- []*kolideclient.Device{device}
		}
	}

	log.Info("bye")
}

func (handler *Handler) UpdateDeviceHealth(db *database.APIServerDB) {
	// TODO
}

const FullSyncInterval = 5 * time.Minute
const FullSyncTimeout = 3 * time.Minute // Must not be greater than FullSyncInterval
func (handler *Handler) Cron(programContext context.Context) {
	ticker := time.NewTicker(time.Second * 1)

	for {
		select {
		case <-ticker.C:
			ticker.Reset(FullSyncInterval)
			log.Info("Doing full Kolide device health sync")
			ctx, cancel := context.WithTimeout(programContext, FullSyncTimeout)
			devices, err := handler.kolideClient.GetDevices(ctx)
			cancel()
			if err != nil {
				log.Errorf("getting devies: %v", err)
			}

			devicesJson, err := json.Marshal(devices)
			if err != nil {
				log.Errorf("marshal json: %v", err)
			}

			log.Infof("%s", devicesJson)

		case <-programContext.Done():
			log.Infof("stopping cron")
			return
		}
	}
}
