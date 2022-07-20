package device_agent

import (
	"context"
	"fmt"
	"os"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/nais/device/pkg/outtune"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
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

func (das *DeviceAgentServer) GetTenants(ctx context.Context, req *pb.GetTenantsRequest) (*pb.GetTenantsResponse, error) {
	bucketName := os.Getenv("NAISDEVICE_TENANTS_BUCKET")
	if bucketName == "" {
		return nil, status.Errorf(codes.PermissionDenied, "NAISDEVICE_TENANTS_BUCKET not set")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	bucket := client.Bucket(bucketName)

	objs := bucket.Objects(ctx, &storage.Query{})

	var tenants []*pb.Tenant
	for {
		obj, err := objs.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return nil, err
		}

		if obj == nil {
			break
		}

		tenants = append(tenants, &pb.Tenant{
			Id:   obj.Name,
			Name: obj.Name,
		})
	}

	return &pb.GetTenantsResponse{
		Tenants: tenants,
	}, nil
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
