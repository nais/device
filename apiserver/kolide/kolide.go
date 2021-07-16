package kolide

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/nais/device/apiserver/database"
	kolideclient "github.com/nais/device/pkg/kolide-client"
	"strings"
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
	checkDevices chan<- []*kolideclient.Device
	db           *database.APIServerDB
}

func New(kolideApiToken string, db *database.APIServerDB) *Handler {
	return &Handler{
		kolideClient: kolideclient.New(kolideApiToken),
		checkDevices: make(chan []*kolideclient.Device, 50),
		db:           db,
	}
}

func (handler *Handler) DeviceEventHandler(ctx context.Context, grpcAddress, grpcToken string) {
	interceptor := &ClientInterceptor{
		RequireTLS: false,
		Token:      grpcToken,
	}

	cred := credentials.NewTLS(&tls.Config{})
	conn, err := grpc.DialContext(ctx, grpcAddress, grpc.WithTransportCredentials(cred), grpc.WithPerRPCCredentials(interceptor))
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

			err = handler.updateDeviceHealth(ctx, device)
			if err != nil {
				log.Warningf("update device health: %v", err)
			}

			handler.checkDevices <- []*kolideclient.Device{device}
		}
	}

	log.Info("bye")
}

const FullSyncInterval = 5 * time.Minute
const FullSyncTimeout = 4 * time.Minute // Must not be greater than FullSyncInterval
func (handler *Handler) Cron(programContext context.Context) {
	ticker := time.NewTicker(time.Second * 1)

	for {
		select {
		case <-ticker.C:
			ticker.Reset(FullSyncInterval)
			log.Info("Doing full Kolide device health sync")
			ctx, cancel := context.WithTimeout(programContext, FullSyncTimeout)
			devices, err := handler.kolideClient.GetDevices(ctx)
			if err != nil {
				log.Errorf("getting devies: %v", err)
			}

			for _, d := range devices {
				err := handler.updateDeviceHealth(ctx, d)
				if err != nil {
					log.Errorf("update device health: %v", err)
				}
			}
			cancel()

		case <-programContext.Done():
			log.Infof("stopping cron")
			return
		}
	}
}

func (handler *Handler) updateDeviceHealth(ctx context.Context, device *kolideclient.Device) error {
	existingDevice, err := handler.db.ReadDeviceBySerialPlatformUsername(ctx, device.Serial, platform(device.Platform), device.AssignedOwner.Email)
	if err != nil {
		return fmt.Errorf("read device(%s, %s): %w", device.AssignedOwner.Email, device.Serial, err)
	}

	existingDevice.Healthy = boolp(DeviceHealthy(device))
	existingDevice.KolideLastSeen = int64p(device.LastSeenAt.Unix())

	err = handler.db.UpdateDevices(ctx, []database.Device{*existingDevice})
	if err != nil {
		return fmt.Errorf("update device: %w", err)
	}

	return nil
}

func DeviceHealthy(device *kolideclient.Device) bool {
	healthy := true

	for _, failure := range device.Failures {
		if failure == nil || failure.Ignored || failure.ResolvedAt != nil {
			continue
		}

		if kolideclient.AfterGracePeriod(*failure) {
			healthy = false
			break
		}
	}

	return healthy
}

func platform(platform string) string {
	switch strings.ToLower(platform) {
	case "darwin":
		return "darwin"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

func boolp(b bool) *bool {
	return &b
}

func int64p(i int64) *int64 {
	return &i
}
