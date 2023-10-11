package integrationtest_test

import (
	"context"
	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/helper"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpc_status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc/test/bufconn"
)

const BufConnSize = 1024 * 1024

func ContextBufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	db := NewDB(t)

	apiserverGRPCServer := NewAPIServer(t, ctx, db)
	apiserverListener := bufconn.Listen(BufConnSize)
	go func() {
		err := apiserverGRPCServer.Serve(apiserverListener)
		if err != nil {
			t.Fatalf("failed to serve apiserver: %v", err)
		}
	}()

	osConfigurator := helper.NewMockOSConfigurator(t)
	osConfigurator.EXPECT().SetupInterface(mock.Anything, mock.Anything).Return(nil)
	osConfigurator.EXPECT().SyncConf(mock.Anything, mock.Anything).Return(nil)
	osConfigurator.EXPECT().SetupRoutes(mock.Anything, mock.Anything).Return(nil)

	helperGRPCServer := NewHelper(t, osConfigurator)
	helperListener := bufconn.Listen(BufConnSize)
	go func() {
		err := helperGRPCServer.Serve(helperListener)
		if err != nil {
			t.Fatalf("failed to serve helper: %v", err)
		}
	}()

	apiDial, err := testDial(ctx, apiserverListener)
	assert.NoError(t, err)

	rc := runtimeconfig.NewMockRuntimeConfig(t)
	rc.EXPECT().DialAPIServer(mock.Anything).Return(apiDial, nil).Once()
	rc.EXPECT().Tenants().Return([]*pb.Tenant{{
		Name:         "test",
		AuthProvider: pb.AuthProvider_Google,
		Domain:       "test.nais.io",
	}})
	rc.EXPECT().SetToken(mock.Anything).Return()
	rc.EXPECT().ResetEnrollConfig().Return()
	rc.EXPECT().GetTenantSession().Return(&pb.Session{
		Key:    "test_key",
		Expiry: timestamppb.New(time.Now().Add(time.Hour)),
	}, nil)
	rc.EXPECT().LoadEnrollConfig().Return(nil)
	peers := &pb.Gateway{
		PublicKey: "test_public_key",
		Endpoint:  "test_endpoint",
		Ipv4:      "192.0.2.10",
	}
	rc.EXPECT().APIServerPeer().Return(peers)
	rc.EXPECT().BuildHelperConfiguration([]*pb.Gateway{peers}).Return(&pb.Configuration{
		Gateways: []*pb.Gateway{peers},
	})

	deviceAgentGRPC := NewDeviceAgent(t, ctx, helperListener, rc)
	deviceAgentListener := bufconn.Listen(BufConnSize)
	go func() {
		err := deviceAgentGRPC.Serve(deviceAgentListener)
		if err != nil {
			t.Fatalf("failed to serve device agent: %v", err)
		}
	}()

	deviceAgentConnection, err := testDial(ctx, deviceAgentListener)
	assert.NoError(t, err)

	deviceAgentClient := pb.NewDeviceAgentClient(deviceAgentConnection)
	statusClient, err := deviceAgentClient.Status(ctx, &pb.AgentStatusRequest{})
	assert.NoError(t, err)

	_, err = deviceAgentClient.Login(ctx, &pb.LoginRequest{})
	assert.NoError(t, err)

	statusChan := make(chan *pb.AgentStatus)
	go func() {
		for {
			status, err := statusClient.Recv()
			if grpc_status.Code(err) == codes.Canceled {
				return
			}
			assert.NoError(t, err)
			statusChan <- status
		}
	}()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timed out without agent reacing connected state")
		case status := <-statusChan:
			if status.ConnectionState == pb.AgentState_Connected {
				return
			}
		}
	}
}

func testDial(ctx context.Context, deviceAgentConn *bufconn.Listener) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(ContextBufDialer(deviceAgentConn)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}
