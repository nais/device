package device_agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	multierror "github.com/hashicorp/go-multierror"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/pb"
)

type DeviceAgentServer struct {
	pb.UnimplementedDeviceAgentServer
	AgentStatus  *pb.AgentStatus
	DeviceHelper pb.DeviceHelperClient
	lock         sync.Mutex
	stateChange  chan pb.AgentState
	streams      map[uuid.UUID]pb.DeviceAgent_StatusServer
	Config       *config.Config
	rc           *runtimeconfig.RuntimeConfig
}

func (das *DeviceAgentServer) Login(ctx context.Context, request *pb.LoginRequest) (*pb.LoginResponse, error) {
	das.stateChange <- pb.AgentState_Authenticating
	return &pb.LoginResponse{}, nil
}

func (das *DeviceAgentServer) Logout(ctx context.Context, request *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	das.stateChange <- pb.AgentState_Disconnecting
	return &pb.LogoutResponse{}, nil
}

func (das *DeviceAgentServer) Status(request *pb.AgentStatusRequest, statusServer pb.DeviceAgent_StatusServer) error {
	id := uuid.New()

	log.Infof("grpc: client connection established")

	das.lock.Lock()
	das.streams[id] = statusServer
	das.lock.Unlock()

	defer func() {
		log.Infof("grpc: client connection closed")
		if !request.GetKeepConnectionOnComplete() {
			log.Infof("grpc: keepalive not requested, tearing down connections...")
			das.stateChange <- pb.AgentState_Disconnecting
		}
		das.lock.Lock()
		delete(das.streams, id)
		das.lock.Unlock()
	}()

	err := statusServer.Send(das.AgentStatus)
	if err != nil {
		return err
	}

	<-statusServer.Context().Done()

	return nil
}

func (das *DeviceAgentServer) BroadcastAgentStatus(agentStatus *pb.AgentStatus) error {
	var errors *multierror.Error

	das.lock.Lock()
	for _, stream := range das.streams {
		err := stream.Send(agentStatus)
		if err != nil {
			errors = multierror.Append(errors, fmt.Errorf("forwarding agentStatus: %v", err))
		}
	}
	das.lock.Unlock()

	//goland:noinspection ALL
	return errors.ErrorOrNil()
}

func (das *DeviceAgentServer) ConfigureJITA(context.Context, *pb.ConfigureJITARequest) (*pb.ConfigureJITAResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConfigureJITA not implemented")
}

func (das *DeviceAgentServer) UpdateAgentStatus(status *pb.AgentStatus) {
	das.AgentStatus = status

	err := das.BroadcastAgentStatus(das.AgentStatus)
	if err != nil {
		log.Errorf("while broadcasting agent status")
	}
}

func (das *DeviceAgentServer) SetAgentConfiguration(ctx context.Context, req *pb.SetAgentConfigurationRequest) (*pb.SetAgentConfigurationResponse, error) {
	log.Infof("setting agent config to: %+v", *req.Config)
	das.Config.AgentConfiguration = req.Config
	das.Config.PersistAgentConfiguration()
	das.stateChange <- pb.AgentState_AgentConfigurationChanged
	return &pb.SetAgentConfigurationResponse{}, nil
}

func (das *DeviceAgentServer) GetAgentConfiguration(ctx context.Context, req *pb.GetAgentConfigurationRequest) (*pb.GetAgentConfigurationResponse, error) {
	log.Infof("returning agent config: %+v", *das.Config.AgentConfiguration)
	return &pb.GetAgentConfigurationResponse{
		Config: das.Config.AgentConfiguration,
	}, nil
}

func NewServer(helper pb.DeviceHelperClient, cfg *config.Config, rc *runtimeconfig.RuntimeConfig) *DeviceAgentServer {
	return &DeviceAgentServer{
		DeviceHelper: helper,
		stateChange:  make(chan pb.AgentState, 32),
		streams:      make(map[uuid.UUID]pb.DeviceAgent_StatusServer),
		Config:       cfg,
		rc:           rc,
	}
}
