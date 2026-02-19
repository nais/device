// Package deviceagent handles the gRPC server that ties the helper, systray/cli and apiserver together. It is the main driver on the device side.
package deviceagent

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/nais/device/internal/deviceagent/acceptableuse"
	"github.com/nais/device/internal/deviceagent/agenthttp"
	"github.com/nais/device/internal/deviceagent/auth"
	"github.com/nais/device/internal/deviceagent/jita"
	"github.com/nais/device/internal/deviceagent/open"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nais/device/internal/notify"

	"github.com/nais/device/internal/deviceagent/config"
	"github.com/nais/device/internal/deviceagent/runtimeconfig"
	"github.com/nais/device/internal/deviceagent/statemachine/state"
	"github.com/nais/device/pkg/pb"
)

type DeviceAgentServer struct {
	pb.UnimplementedDeviceAgentServer
	Config    *config.Config
	notifier  notify.Notifier
	rc        runtimeconfig.RuntimeConfig
	log       *logrus.Entry
	sendEvent func(state.EventWithSpan)

	statusChannelsLock sync.RWMutex
	statusChannels     map[uuid.UUID]chan *pb.AgentStatus

	AgentStatus     *pb.AgentStatus
	agentStatusLock sync.RWMutex

	acceptaleUseHandler *acceptableuse.Handler
	jitaHandler         *jita.Handler
	authHandler         auth.Handler
}

func (das *DeviceAgentServer) Login(ctx context.Context, request *pb.LoginRequest) (*pb.LoginResponse, error) {
	das.sendEvent(state.SpanEvent(ctx, state.EventLogin))
	return &pb.LoginResponse{}, nil
}

func (das *DeviceAgentServer) Logout(ctx context.Context, request *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	das.sendEvent(state.SpanEvent(ctx, state.EventDisconnect))
	return &pb.LogoutResponse{}, nil
}

func (das *DeviceAgentServer) Status(request *pb.AgentStatusRequest, statusServer pb.DeviceAgent_StatusServer) error {
	id := uuid.New()

	das.log.Debug("grpc: client connection established to device helper")

	agentStatusChan := make(chan *pb.AgentStatus, 8)
	das.agentStatusLock.RLock()
	agentStatusChan <- das.AgentStatus
	das.agentStatusLock.RUnlock()

	das.statusChannelsLock.Lock()
	das.statusChannels[id] = agentStatusChan
	das.statusChannelsLock.Unlock()

	defer func() {
		das.log.Debug("grpc: client connection with device helper closed")
		das.agentStatusLock.RLock()
		keepConnection := request.GetKeepConnectionOnComplete()
		connectionState := das.AgentStatus.GetConnectionState()
		das.agentStatusLock.RUnlock()
		if !keepConnection || connectionState != pb.AgentState_Connected {
			das.log.Debug("grpc: keepalive not requested, tearing down connections...")
			das.sendEvent(state.SpanEvent(statusServer.Context(), state.EventDisconnect))
		}
		das.statusChannelsLock.Lock()
		close(agentStatusChan)
		delete(das.statusChannels, id)
		das.statusChannelsLock.Unlock()
	}()

	for {
		select {
		case <-statusServer.Context().Done():
			return nil
		case status := <-agentStatusChan:
			err := statusServer.Send(status)
			if err != nil {
				das.log.WithError(err).Error("while sending agent status")
			}
		}
	}
}

func (das *DeviceAgentServer) ConfigureJITA(context.Context, *pb.ConfigureJITARequest) (*pb.ConfigureJITAResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConfigureJITA not implemented")
}

func (das *DeviceAgentServer) UpdateAgentStatus(status *pb.AgentStatus) {
	das.agentStatusLock.Lock()
	das.AgentStatus = status
	das.agentStatusLock.Unlock()

	das.statusChannelsLock.RLock()
	for _, c := range das.statusChannels {
		select {
		case c <- status:
		default:
			das.log.Error("BUG: update agent status: channel is full")
		}
	}
	das.statusChannelsLock.RUnlock()
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
		das.sendEvent(state.SpanEvent(ctx, state.EventDisconnect))
		return &pb.SetActiveTenantResponse{}, nil
	}

	das.sendEvent(state.SpanEvent(ctx, state.EventDisconnect))
	das.log.WithField("name", req.Name).Info("activated tenant")
	return &pb.SetActiveTenantResponse{}, nil
}

func (das *DeviceAgentServer) ShowAcceptableUse(ctx context.Context, _ *pb.ShowAcceptableUseRequest) (*pb.ShowAcceptableUseResponse, error) {
	open.Open(agenthttp.Path("/acceptableUse", true))
	return &pb.ShowAcceptableUseResponse{}, nil
}

func (das *DeviceAgentServer) ShowJita(ctx context.Context, req *pb.ShowJitaRequest) (*pb.ShowJitaResponse, error) {
	gateway := req.Gateway
	url := agenthttp.Path("/jita?gateway="+gateway, true)

	tok := das.rc.GetJitaToken(ctx)
	if tok.Valid() {
		open.Open(url)
		return &pb.ShowJitaResponse{}, nil
	}

	tokens, err := das.authHandler.GetDeviceAgentToken(ctx, das.log, das.Config.JitaOAuth2Config, url)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unable to get JITA token from Entra: %v", err)
	}

	if !tokens.Token.Valid() {
		return nil, fmt.Errorf("received invalid JITA token")
	}

	das.rc.SetJitaToken(tokens.Token)
	return &pb.ShowJitaResponse{}, nil
}

func NewServer(ctx context.Context,
	log *logrus.Entry,
	cfg *config.Config,
	rc runtimeconfig.RuntimeConfig,
	notifier notify.Notifier,
	sendEvent func(state.EventWithSpan),
	acceptableUse *acceptableuse.Handler,
	jita *jita.Handler,
	authHandler auth.Handler,
) *DeviceAgentServer {
	return &DeviceAgentServer{
		log:                 log,
		AgentStatus:         &pb.AgentStatus{ConnectionState: pb.AgentState_Disconnected},
		statusChannels:      make(map[uuid.UUID]chan *pb.AgentStatus),
		Config:              cfg,
		rc:                  rc,
		notifier:            notifier,
		sendEvent:           sendEvent,
		acceptaleUseHandler: acceptableUse,
		jitaHandler:         jita,
		authHandler:         authHandler,
	}
}
