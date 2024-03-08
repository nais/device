package device_agent

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nais/device/internal/notify"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine"
	"github.com/nais/device/internal/pb"
)

type DeviceAgentServer struct {
	pb.UnimplementedDeviceAgentServer
	AgentStatus    *pb.AgentStatus
	lock           sync.RWMutex
	statusChannels map[uuid.UUID]chan *pb.AgentStatus
	Config         *config.Config
	notifier       notify.Notifier
	rc             runtimeconfig.RuntimeConfig
	log            *logrus.Entry
	sendEvent      func(statemachine.Event)
}

func (das *DeviceAgentServer) Login(ctx context.Context, request *pb.LoginRequest) (*pb.LoginResponse, error) {
	das.sendEvent(statemachine.EventLogin)
	return &pb.LoginResponse{}, nil
}

func (das *DeviceAgentServer) Logout(ctx context.Context, request *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	das.sendEvent(statemachine.EventDisconnect)
	return &pb.LogoutResponse{}, nil
}

func (das *DeviceAgentServer) Status(request *pb.AgentStatusRequest, statusServer pb.DeviceAgent_StatusServer) error {
	id := uuid.New()

	das.log.Debug("grpc: client connection established to device helper")

	agentStatusChan := make(chan *pb.AgentStatus, 8)
	agentStatusChan <- das.AgentStatus

	das.lock.Lock()
	das.statusChannels[id] = agentStatusChan
	das.lock.Unlock()

	defer func() {
		das.log.Debugf("grpc: client connection with device helper closed")
		if !request.GetKeepConnectionOnComplete() {
			das.log.Debugf("grpc: keepalive not requested, tearing down connections...")
			das.sendEvent(statemachine.EventDisconnect)
		}
		close(agentStatusChan)
		das.lock.Lock()
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
				das.log.Errorf("while sending agent status: %s", err)
			}
		}
	}
}

func (das *DeviceAgentServer) ConfigureJITA(context.Context, *pb.ConfigureJITARequest) (*pb.ConfigureJITAResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConfigureJITA not implemented")
}

func (das *DeviceAgentServer) UpdateAgentStatus(status *pb.AgentStatus) {
	das.AgentStatus = status

	das.lock.RLock()
	for _, c := range das.statusChannels {
		select {
		case c <- status:
		default:
			das.log.Errorf("BUG: update agent status: channel is full")
		}
	}
	das.lock.RUnlock()
}

func (das *DeviceAgentServer) SetAgentConfiguration(ctx context.Context, req *pb.SetAgentConfigurationRequest) (*pb.SetAgentConfigurationResponse, error) {
	das.Config.AgentConfiguration = req.Config
	das.Config.PersistAgentConfiguration(das.log)
	return &pb.SetAgentConfigurationResponse{}, nil
}

func (das *DeviceAgentServer) GetAgentConfiguration(ctx context.Context, req *pb.GetAgentConfigurationRequest) (*pb.GetAgentConfigurationResponse, error) {
	return &pb.GetAgentConfigurationResponse{
		Config: das.Config.AgentConfiguration,
	}, nil
}

func (das *DeviceAgentServer) SetActiveTenant(ctx context.Context, req *pb.SetActiveTenantRequest) (*pb.SetActiveTenantResponse, error) {
	if err := das.rc.SetActiveTenant(req.Name); err != nil {
		das.notifier.Errorf("while activating tenant: %s", err)
		das.sendEvent(statemachine.EventDisconnect)
		return &pb.SetActiveTenantResponse{}, nil
	}

	das.sendEvent(statemachine.EventDisconnect)
	das.log.Infof("activated tenant: %s", req.Name)
	return &pb.SetActiveTenantResponse{}, nil
}

func NewServer(ctx context.Context,
	log *logrus.Entry,
	cfg *config.Config,
	rc runtimeconfig.RuntimeConfig,
	notifier notify.Notifier,
	sendEvent func(statemachine.Event),
) *DeviceAgentServer {
	return &DeviceAgentServer{
		log:            log,
		AgentStatus:    &pb.AgentStatus{ConnectionState: pb.AgentState_Disconnected},
		statusChannels: make(map[uuid.UUID]chan *pb.AgentStatus),
		Config:         cfg,
		rc:             rc,
		notifier:       notifier,
		sendEvent:      sendEvent,
	}
}
