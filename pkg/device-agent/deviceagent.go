package device_agent

import (
	"context"

	pb "github.com/nais/device/pkg/protobuf"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DeviceAgentServer struct {
	pb.UnimplementedDeviceAgentServer
	stateChange chan ProgramState
}

func (d DeviceAgentServer) Login(ctx context.Context, request *pb.LoginRequest) (*pb.LoginResponse, error) {
	d.stateChange <- StateBootstrapping
	return &pb.LoginResponse{}, nil
}

func (d DeviceAgentServer) Logout(ctx context.Context, request *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	d.stateChange <- StateDisconnecting
	return &pb.LogoutResponse{}, nil
}

func (d DeviceAgentServer) Status(*pb.AgentStatusRequest, pb.DeviceAgent_StatusServer) error {
	return status.Errorf(codes.Unimplemented, "method Status not implemented")
}

func (d DeviceAgentServer) ConfigureJITA(context.Context, *pb.ConfigureJITARequest) (*pb.ConfigureJITAResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConfigureJITA not implemented")
}

func NewServer(stateChange chan ProgramState) pb.DeviceAgentServer {
	return &DeviceAgentServer{
		stateChange: stateChange,
	}
}
