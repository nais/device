package api

import (
	"context"
	"fmt"
	"github.com/nais/device/apiserver/database"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"sync"
)

type grpcServer struct {
	pb.UnimplementedAPIServerServer

	streams map[string]pb.APIServer_GetDeviceConfigurationServer
	lock    sync.Mutex
	db      *database.APIServerDB
}

var _ pb.APIServerServer = &grpcServer{}

func NewGRPCServer(db *database.APIServerDB) *grpcServer {
	return &grpcServer{
		streams: make(map[string]pb.APIServer_GetDeviceConfigurationServer),
		db:      db,
	}
}

func (s *grpcServer) GetDeviceConfiguration(request *pb.GetDeviceConfigurationRequest, stream pb.APIServer_GetDeviceConfigurationServer) error {
	s.lock.Lock()
	s.streams[request.SessionKey] = stream
	s.lock.Unlock()

	// send initial device configuration
	err := s.SendDeviceConfiguration(stream.Context(), request.SessionKey)
	if err != nil {
		log.Errorf("send initial device configuration: %s", err)
	}

	// wait for disconnect
	<-stream.Context().Done()

	s.lock.Lock()
	delete(s.streams, request.SessionKey)
	s.lock.Unlock()

	return nil
}

func (s *grpcServer) SendDeviceConfiguration(ctx context.Context, sessionKey string) error {
	log.Infof("SendDeviceConfiguration(%s)", sessionKey)

	stream, ok := s.streams[sessionKey]
	if !ok {
		return fmt.Errorf("no session")
	}

	sessionInfo, err := s.db.ReadSessionInfo(ctx, sessionKey)
	if err != nil {
		return err
	}

	device, err := s.db.ReadDeviceById(ctx, sessionInfo.Device.ID)
	if err != nil {
		return fmt.Errorf("read device from db: %v", err)
	}

	if device.Healthy == nil || !*device.Healthy {
		return stream.Send(&pb.GetDeviceConfigurationResponse{
			Status: pb.DeviceConfigurationStatus_DeviceUnhealthy,
		})
	}

	gateways, err := s.UserGateways(sessionInfo.Groups)

	m, err := DeviceConfigsReturned.GetMetricWithLabelValues(device.Serial, device.Username)
	if err != nil {
		log.Errorf("BUG: get metric: %s", err)
	} else {
		m.Inc()
	}

	return stream.Send(&pb.GetDeviceConfigurationResponse{
		Status:   pb.DeviceConfigurationStatus_DeviceHealthy,
		Gateways: gateways,
	})
}

func (s *grpcServer) UserGateways(userGroups []string) ([]*pb.Gateway, error) {
	gateways, err := s.db.ReadGateways()
	if err != nil {
		return nil, fmt.Errorf("reading gateways from db: %v", err)
	}

	var filtered []*pb.Gateway
	for _, gw := range gateways {
		if userIsAuthorized(gw.AccessGroupIDs, userGroups) {
			filtered = append(filtered, gw)
		}
	}

	return filtered, nil
}
