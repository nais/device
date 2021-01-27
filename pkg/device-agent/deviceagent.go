package device_agent

import (
	"context"
	pb "github.com/nais/device/pkg/protobuf"
	"github.com/prometheus/common/log"
)

type DeviceAgentServer struct {
	pb.UnimplementedDeviceAgentServer
	stateChange chan ProgramState
}

func (d DeviceAgentServer) Connect(ctx context.Context, empty *pb.Empty) (*pb.Error, error) {
	d.stateChange <- StateBootstrapping
	return nil, nil
}

func (d DeviceAgentServer) Disconnect(ctx context.Context, empty *pb.Empty) (*pb.Error, error) {
	d.stateChange <- StateDisconnecting
	return nil, nil
}

func (d DeviceAgentServer) WatchGateways(empty *pb.Empty, server pb.DeviceAgent_WatchGatewaysServer) error {
	log.Error("not implemented")
	return nil
}

func (d DeviceAgentServer) GatewayClicked(ctx context.Context, gateway *pb.Gateway) (*pb.Error, error) {
	log.Error("not implemented")
	return nil, nil
}


func NewServer(stateChange chan ProgramState) pb.DeviceAgentServer {
	return &DeviceAgentServer{
		stateChange: stateChange,
	}
}
