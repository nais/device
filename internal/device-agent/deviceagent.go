// Package device_agent handles the gRPC server that ties the helper, systray/cli and apiserver together. It is the main driver on the device side.
package device_agent

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nais/device/internal/device-agent/acceptableuse"
	"github.com/nais/device/internal/device-agent/open"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nais/device/internal/notify"

	"github.com/nais/device/internal/device-agent/config"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine/state"
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
		if !request.GetKeepConnectionOnComplete() {
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
	return nil, das.rc.WithAPIServer(func(apiserver pb.APIServerClient, key string) error {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return status.Errorf(codes.FailedPrecondition, "while creating acceptable use listener: %v", err)
		}
		url := "http://" + listener.Addr().String()
		das.log.WithField("url", url).Debug("acceptable use server listening")

		// TODO: fetch from apiserver
		hasApproved := false

		toggleApproval := make(chan struct{})
		keepAlive := make(chan struct{})

		srv := acceptableuse.NewServer(hasApproved, toggleApproval, keepAlive)
		go func() {
			if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				das.log.WithError(err).Error("while serving acceptable use")
			}
		}()

		defer func() { _ = srv.Close() }()

		open.Open(url)

		for {
			select {
			case <-toggleApproval:
				// TODO: call apiserver to toggle approval

				das.log.Info("closing acceptable use server")
				return nil
			case <-keepAlive:
				das.log.Info("keeping acceptable use server alive")
			case <-time.After(10 * time.Second):
				das.log.Info("closing acceptable use server")
				return nil
			case <-ctx.Done():
				return nil
			}
		}
	})
}

func NewServer(ctx context.Context,
	log *logrus.Entry,
	cfg *config.Config,
	rc runtimeconfig.RuntimeConfig,
	notifier notify.Notifier,
	sendEvent func(state.EventWithSpan),
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
