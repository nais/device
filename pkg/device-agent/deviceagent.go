package device_agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	pb "github.com/nais/device/pkg/protobuf"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeviceAgentServer struct {
	pb.UnimplementedDeviceAgentServer
	stateChange  chan ProgramState
	statusChange chan *pb.AgentStatus
	streams      map[uuid.UUID]pb.DeviceAgent_StatusServer
	AgentStatus  pb.AgentStatus
	lock         sync.Mutex
}

func (das *DeviceAgentServer) Login(ctx context.Context, request *pb.LoginRequest) (*pb.LoginResponse, error) {
	das.stateChange <- StateBootstrapping
	return &pb.LoginResponse{}, nil
}

func (das *DeviceAgentServer) Logout(ctx context.Context, request *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	das.stateChange <- StateDisconnecting
	return &pb.LogoutResponse{}, nil
}

func (das *DeviceAgentServer) Status(request *pb.AgentStatusRequest, statusServer pb.DeviceAgent_StatusServer) error {
	id := uuid.New()

	das.lock.Lock()
	das.streams[id] = statusServer
	das.lock.Unlock()

	<-statusServer.Context().Done()

	das.lock.Lock()
	delete(das.streams, id)
	das.lock.Unlock()

	return nil
}

func (das *DeviceAgentServer) BroadcastAgentStatus(agentStatus *pb.AgentStatus) error {
	var errors *multierror.Error
	for _, stream := range das.streams {
		err := stream.Send(agentStatus)
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("forwarding agentStatus: %v", err))
		}
	}

	return errors.ErrorOrNil()
}

func (das *DeviceAgentServer) ConfigureJITA(context.Context, *pb.ConfigureJITARequest) (*pb.ConfigureJITAResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConfigureJITA not implemented")
}

func (das *DeviceAgentServer) UpdateAgentStatusGateways(gateways []*pb.Gateway) {
	das.AgentStatus.Gateways = gateways

	err := das.BroadcastAgentStatus(&das.AgentStatus)
	if err != nil {
		log.Errorf("while broadcasting agent status")
	}
}

func NewServer(stateChange chan ProgramState) *DeviceAgentServer {
	return &DeviceAgentServer{
		stateChange: stateChange,
		streams:     make(map[uuid.UUID]pb.DeviceAgent_StatusServer, 0),
	}
}