package integrationtest_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nais/device/pkg/device-agent/runtimeconfig"
	"github.com/nais/device/pkg/helper"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpc_status "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/grpc/test/bufconn"
)

func bufconnDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func serve(server *grpc.Server, errs chan<- error) (*bufconn.Listener, func()) {
	lis := bufconn.Listen(1024 * 1024)
	go func() {
		errs <- server.Serve(lis)
	}()

	return lis, server.Stop
}

func dial(ctx context.Context, lis *bufconn.Listener) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufconnDialer(lis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
}

func TestIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	logrus.SetLevel(logrus.DebugLevel)

	testDevice := &pb.Device{
		Serial:      "test-serial",
		LastUpdated: &timestamppb.Timestamp{},
		Healthy:     true,
		PublicKey:   "publicKey",
		Ipv4:        "192.0.2.20",
		Username:    "tester",
		Platform:    "linux",
		Ipv6:        "2001:db8::20",
	}

	db := NewDB(t)
	assert.NoError(t, db.AddDevice(ctx, testDevice))
	// populate device with data from db, specifically to get the ID
	testDevice, err := db.ReadDevice(ctx, testDevice.PublicKey)
	assert.NoError(t, err)

	session := &pb.Session{
		Key:      "session-key",
		Expiry:   timestamppb.New(time.Now().Add(24 * time.Hour)),
		Device:   testDevice,
		Groups:   []string{"test-group"},
		ObjectID: "test-object-id",
	}
	assert.NoError(t, db.AddSessionInfo(ctx, session))

	sessions, err := db.ReadSessionInfos(ctx)
	assert.NoError(t, err)
	t.Logf("sessions: %+v ", sessions)

	serveErrs := make(chan error)

	apiserverListener, cancel := serve(NewAPIServer(t, ctx, db), serveErrs)
	defer cancel()

	osConfigurator := helper.NewMockOSConfigurator(t)
	osConfigurator.EXPECT().SetupInterface(mock.Anything, mock.Anything).Return(nil)
	osConfigurator.EXPECT().SyncConf(mock.Anything, mock.Anything).Return(nil)
	osConfigurator.EXPECT().SetupRoutes(mock.Anything, mock.Anything).Return(nil)

	helperListener, cancel := serve(NewHelper(t, osConfigurator), serveErrs)
	defer cancel()

	apiDial, err := dial(ctx, apiserverListener)
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
	rc.EXPECT().GetTenantSession().Return(session, nil)
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

	deviceAgentListener, cancel := serve(NewDeviceAgent(t, ctx, helperListener, rc), serveErrs)
	defer cancel()

	deviceAgentConnection, err := dial(ctx, deviceAgentListener)
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
			if grpc_status.Code(err) == codes.Canceled || grpc_status.Code(err) == codes.Unavailable {
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
		case err := <-serveErrs:
			if err != nil {
				t.Fatalf("grpc serve error: %v", err)
			}
		case status := <-statusChan:
			if status.ConnectionState == pb.AgentState_Connected {
				return
			}
		}
	}
}
