package device_agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeviceAgentServer struct {
	pb.UnimplementedDeviceAgentServer
	AgentStatus                       *pb.AgentStatus
	DeviceHelper                      pb.DeviceHelperClient
	lock                              sync.Mutex
	stateChange                       chan pb.AgentState
	statusChange                      chan *pb.AgentStatus
	streams                           map[uuid.UUID]pb.DeviceAgent_StatusServer
	enableMicrosoftCertificateRenewal bool
}

func (das *DeviceAgentServer) Login(ctx context.Context, request *pb.LoginRequest) (*pb.LoginResponse, error) {
	das.stateChange <- pb.AgentState_Bootstrapping
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

func (das *DeviceAgentServer) EnableClientCertRenewal(ctx context.Context, req *pb.EnableCertRenewalRequest) (*pb.EnableCertRenewalResponse, error) {
	if req.Enable {
		das.stateChange <-pb.AgentState_EnableClientCertRenewal
	} else {
		das.stateChange <-pb.AgentState_DisableClientCertRenewal
	}
	return &pb.EnableCertRenewalResponse{}, nil
}

func NewServer(helper pb.DeviceHelperClient, enableMicrosoftCertificateRenewal bool) *DeviceAgentServer {
	return &DeviceAgentServer{
		DeviceHelper: helper,
		stateChange:  make(chan pb.AgentState, 32),
		streams:      make(map[uuid.UUID]pb.DeviceAgent_StatusServer, 0),
		enableMicrosoftCertificateRenewal: enableMicrosoftCertificateRenewal,
	}
}
