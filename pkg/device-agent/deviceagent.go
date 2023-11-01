package device_agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nais/device/pkg/notify"

	"github.com/nais/device/pkg/device-agent/config"
	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/pb"
)

type DeviceAgentServer struct {
	pb.UnimplementedDeviceAgentServer
	AgentStatus    *pb.AgentStatus
	DeviceHelper   pb.DeviceHelperClient
	lock           sync.Mutex
	stateChange    chan pb.AgentState
	statusChannels map[uuid.UUID]chan *pb.AgentStatus
	Config         *config.Config
	rc             runtimeconfig.RuntimeConfig
	notifier       notify.Notifier
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

	agentStatusChan := make(chan *pb.AgentStatus, 1)
	agentStatusChan <- das.AgentStatus

	das.lock.Lock()
	das.statusChannels[id] = agentStatusChan
	das.lock.Unlock()

	defer func() {
		log.Debugf("grpc: client connection with device helper closed")
		if !request.GetKeepConnectionOnComplete() {
			log.Debugf("grpc: keepalive not requested, tearing down connections...")
			das.stateChange <- pb.AgentState_Disconnecting
		}
		das.lock.Lock()
		close(agentStatusChan)
		delete(das.statusChannels, id)
		das.lock.Unlock()
	}()

	for {
		select {
		case <-statusServer.Context().Done():
			return nil
		case status := <-agentStatusChan:
			err := statusServer.Send(status)
			if err != nil {
				log.Errorf("while sending agent status: %s", err)
			}
		}
	}
}

func (das *DeviceAgentServer) ConfigureJITA(context.Context, *pb.ConfigureJITARequest) (*pb.ConfigureJITAResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConfigureJITA not implemented")
}

func (das *DeviceAgentServer) UpdateAgentStatus(status *pb.AgentStatus) {
	das.AgentStatus = status

	das.lock.Lock()
	for _, c := range das.statusChannels {
		c <- status
	}
	das.lock.Unlock()
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
	if err := das.rc.SetActiveTenant(req.Name); err != nil {
		das.Notifier().Errorf("while activating tenant: %s", err)
		das.stateChange <- pb.AgentState_Disconnecting
		return &pb.SetActiveTenantResponse{}, nil
	}

	das.stateChange <- pb.AgentState_Disconnecting
	log.Infof("activated tenant: %s", req.Name)
	return &pb.SetActiveTenantResponse{}, nil
}

func (das *DeviceAgentServer) Notifier() notify.Notifier {
	return das.notifier
}

func NewServer(helper pb.DeviceHelperClient, cfg *config.Config, rc runtimeconfig.RuntimeConfig, notifier notify.Notifier) *DeviceAgentServer {
	return &DeviceAgentServer{
		DeviceHelper:   helper,
		AgentStatus:    &pb.AgentStatus{ConnectionState: pb.AgentState_Disconnected},
		stateChange:    make(chan pb.AgentState, 32),
		statusChannels: make(map[uuid.UUID]chan *pb.AgentStatus),
		Config:         cfg,
		rc:             rc,
		notifier:       notifier,
	}
}
