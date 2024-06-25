package pb

import (
	"context"

	"github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

func NewMockHelperClient(log logrus.FieldLogger) DeviceHelperClient {
	return &mockHelper{
		log: log,
	}
}

type mockHelper struct {
	log logrus.FieldLogger
}

func (m *mockHelper) Configure(ctx context.Context, in *Configuration, opts ...grpc.CallOption) (*ConfigureResponse, error) {
	m.log.WithField("method", "Configure").Info("mock helper called")
	return &ConfigureResponse{}, nil
}

func (m *mockHelper) Teardown(ctx context.Context, in *TeardownRequest, opts ...grpc.CallOption) (*TeardownResponse, error) {
	m.log.WithField("method", "Teardown").Info("mock helper called")
	return &TeardownResponse{}, nil
}

func (m *mockHelper) Upgrade(ctx context.Context, in *UpgradeRequest, opts ...grpc.CallOption) (*UpgradeResponse, error) {
	m.log.WithField("method", "Upgrade").Info("mock helper called")
	return &UpgradeResponse{}, nil
}

func (m *mockHelper) GetSerial(ctx context.Context, in *GetSerialRequest, opts ...grpc.CallOption) (*GetSerialResponse, error) {
	m.log.WithField("method", "GetSerial").Info("mock helper called")
	return &GetSerialResponse{
		Serial: "mock",
	}, nil
}

func (m *mockHelper) Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingResponse, error) {
	m.log.WithField("method", "Ping").Info("mock helper called")
	return &PingResponse{}, nil
}
