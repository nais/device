package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/config"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/pb"
	"github.com/nais/device/pkg/random"
)

type grpcServer struct {
	pb.UnimplementedAPIServerServer

	store   auth.SessionStore
	streams map[string]pb.APIServer_GetDeviceConfigurationServer
	lock    sync.Mutex
	db      database.APIServer
}

var _ pb.APIServerServer = &grpcServer{}

func NewGRPCServer(db database.APIServer, store auth.SessionStore) *grpcServer {
	return &grpcServer{
		streams: make(map[string]pb.APIServer_GetDeviceConfigurationServer),
		db:      db,
		store:   store,
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

	device, err := s.db.ReadDeviceById(ctx, sessionInfo.GetDevice().GetId())
	if err != nil {
		return fmt.Errorf("read device from db: %v", err)
	}

	if !device.GetHealthy() {
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

func (s *grpcServer) Login(ctx context.Context, request *pb.APIServerLoginRequest) (*pb.APIServerLoginResponse, error) {
	parsedToken, err := jwt.Parse([]byte(request.Token))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, err := parsedToken.AsMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("convert claims to map: %w", err)
	}

	username := claims["preferred_username"].(string)

	var groups []string
	approvalOK := false
	for _, group := range claims["groups"].([]interface{}) {
		s := group.(string)
		if s == config.NaisDeviceApprovalGroup {
			approvalOK = true
		}
		groups = append(groups, s)
	}

	if !approvalOK {
		return nil, fmt.Errorf("do's and don'ts not accepted, visit: https://naisdevice-approval.nais.io/ to read and accept")
	}

	device, err := s.db.ReadDeviceBySerialPlatform(ctx, request.Serial, request.Platform)
	if err != nil {
		return nil, fmt.Errorf("read device (%s, %s), user: %s, err: %v", request.Platform, request.Serial, username, err)
	}

	session := &pb.Session{
		Key:      random.RandomString(20, random.LettersAndNumbers),
		Expiry:   timestamppb.New(time.Now().Add(auth.SessionDuration)),
		Groups:   groups,
		ObjectID: claims["oid"].(string),
		Device:   device,
	}

	err = s.store.Set(ctx, session)
	if err != nil {
		log.Errorf("Persisting session info %v: %v", session, err)
		// don't abort auth here as this might be OK
		// fixme: we must abort auth here as the database didn't accept the session, and further usage will probably fail
		return nil, fmt.Errorf("persist session: %w", err)
	}

	return &pb.APIServerLoginResponse{
		Session: session,
	}, nil
}
