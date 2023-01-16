package device_agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/nais/device/pkg/outtune"
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
	outtune      outtune.Outtune
}

const maxLoginAttempts = 20

func (das *DeviceAgentServer) Login(ctx context.Context, request *pb.LoginRequest) (*pb.LoginResponse, error) {
	var lastStatus pb.AgentState
	for attempt := 1; attempt <= maxLoginAttempts; attempt += 1 {
		lastStatus = das.AgentStatus.ConnectionState
		if lastStatus == pb.AgentState_Disconnected {
			das.stateChange <- pb.AgentState_Authenticating
			return &pb.LoginResponse{}, nil
		}

		log.Debugf("[attempt %d/%d] device agent server login: agent not in correct state (state=%+v). wait 200ms and retry", attempt, maxLoginAttempts, lastStatus)
		time.Sleep(200 * time.Millisecond)
	}

	return &pb.LoginResponse{}, fmt.Errorf("unable to connect, invalid state: %+v", lastStatus)
}

func (das *DeviceAgentServer) Logout(ctx context.Context, request *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	das.stateChange <- pb.AgentState_Disconnecting
	return &pb.LogoutResponse{}, nil
}

func (das *DeviceAgentServer) Status(request *pb.AgentStatusRequest, statusServer pb.DeviceAgent_StatusServer) error {
	id := uuid.New()

	log.Debug("grpc: client connection established to device helper")

	das.lock.Lock()
	das.streams[id] = statusServer
	das.lock.Unlock()

	defer func() {
		log.Debugf("grpc: client connection with device helper closed")
		if !request.GetKeepConnectionOnComplete() {
			log.Debugf("grpc: keepalive not requested, tearing down connections...")
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
	das.Config.AgentConfiguration = req.Config
	das.Config.PersistAgentConfiguration()
	das.stateChange <- pb.AgentState_AgentConfigurationChanged
	return &pb.SetAgentConfigurationResponse{}, nil
}

func (das *DeviceAgentServer) GetAgentConfiguration(ctx context.Context, req *pb.GetAgentConfigurationRequest) (*pb.GetAgentConfigurationResponse, error) {
	return &pb.GetAgentConfigurationResponse{
		Config: das.Config.AgentConfiguration,
	}, nil
}

func (das *DeviceAgentServer) SetActiveTenant(ctx context.Context, req *pb.SetActiveTenantRequest) (*pb.SetActiveTenantResponse, error) {
	// Mark all tenants inactive
	for i := range das.rc.Tenants {
		das.rc.Tenants[i].Active = false
	}

	for i, tenant := range das.rc.Tenants {
		if tenant.Name == req.Name {
			das.rc.Tenants[i].Active = true
			das.stateChange <- pb.AgentState_Disconnecting
			log.Infof("activated tenant: %s", tenant.Name)
			return &pb.SetActiveTenantResponse{}, nil
		}
	}

	das.Notify("Tenant %s not found", req.Name)
	return &pb.SetActiveTenantResponse{}, nil
}

func NewServer(helper pb.DeviceHelperClient, cfg *config.Config, rc *runtimeconfig.RuntimeConfig, ot outtune.Outtune) *DeviceAgentServer {
	return &DeviceAgentServer{
		DeviceHelper: helper,
		stateChange:  make(chan pb.AgentState, 32),
		streams:      make(map[uuid.UUID]pb.DeviceAgent_StatusServer),
		Config:       cfg,
		outtune:      ot,
		rc:           rc,
	}
}
