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

const testGroup = "test-group"

func bufconnDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func serve(t *testing.T, server *grpc.Server) (*bufconn.Listener, func()) {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("grpc serve error: %v", err)
			t.FailNow()
		}
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
	t.Parallel()
	logrus.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name             string
		device           *pb.Device
		endState         pb.AgentState
		expectedGateways []*pb.Gateway
	}{
		{
			name: "test happy unhealthy path",
			device: &pb.Device{
				Serial:    "test-serial",
				Healthy:   false,
				PublicKey: "publicKey",
				Username:  "tester",
				Platform:  "linux",
			},
			endState:         pb.AgentState_Unhealthy,
			expectedGateways: nil,
		},
		{
			name: "test happy healthy path",
			device: &pb.Device{
				Serial:    "test-serial",
				Healthy:   true,
				PublicKey: "publicKey",
				Username:  "tester",
				Platform:  "linux",
			},
			endState: pb.AgentState_Connected,
			expectedGateways: []*pb.Gateway{{
				Name:           "expected-gateway",
				PublicKey:      "gwPublicKey",
				AccessGroupIDs: []string{testGroup},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tableTest(t, test.device, test.endState, test.expectedGateways)
		})
	}
}

func tableTest(t *testing.T, testDevice *pb.Device, endState pb.AgentState, expectedGateways []*pb.Gateway) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	db := NewDB(t)
	assert.NoError(t, db.AddDevice(ctx, testDevice))
	assert.NoError(t, db.UpdateDevices(ctx, []*pb.Device{testDevice}))
	// populate device with data from db, specifically to get the ID
	testDevice, err := db.ReadDevice(ctx, testDevice.PublicKey)
	assert.NoError(t, err)

	for _, gateway := range expectedGateways {
		assert.NoError(t, db.AddGateway(ctx, gateway))
	}

	session := &pb.Session{
		Key:      "session-key",
		Expiry:   timestamppb.New(time.Now().Add(24 * time.Hour)),
		Device:   testDevice,
		Groups:   []string{testGroup},
		ObjectID: "test-object-id",
	}
	assert.NoError(t, db.AddSessionInfo(ctx, session))

	sessions, err := db.ReadSessionInfos(ctx)
	assert.NoError(t, err)
	t.Logf("sessions: %+v ", sessions)

	apiserverListener, stopAPIServer := serve(t, NewAPIServer(t, ctx, db))

	osConfigurator := helper.NewMockOSConfigurator(t)
	osConfigurator.EXPECT().SetupInterface(mock.Anything, mock.Anything).Return(nil)
	osConfigurator.EXPECT().SyncConf(mock.Anything, mock.Anything).Return(nil)
	osConfigurator.EXPECT().SetupRoutes(mock.Anything, mock.Anything).Return(nil)
	osConfigurator.EXPECT().TeardownInterface(mock.Anything).Return(nil).Maybe()

	helperListener, stopHelper := serve(t, NewHelper(t, osConfigurator))

	apiDial, err := dial(ctx, apiserverListener)
	assert.NoError(t, err)

	rc := runtimeconfig.NewMockRuntimeConfig(t)
	rc.EXPECT().DialAPIServer(mock.Anything).Return(apiDial, nil)
	rc.EXPECT().Tenants().Return([]*pb.Tenant{{
		Name:         "test",
		AuthProvider: pb.AuthProvider_Google,
		Domain:       "test.nais.io",
	}})
	rc.EXPECT().SetToken(mock.Anything).Return()
	rc.EXPECT().ResetEnrollConfig().Return()
	rc.EXPECT().GetTenantSession().Return(session, nil)
	rc.EXPECT().LoadEnrollConfig().Return(nil)
	peers := []*pb.Gateway{
		{
			PublicKey: "test_public_key",
			Endpoint:  "test_endpoint",
			Ipv4:      "192.0.2.10",
		},
	}
	rc.EXPECT().APIServerPeer().Return(peers[0])
	rc.EXPECT().BuildHelperConfiguration(mock.MatchedBy(func(gws []*pb.Gateway) bool {
		return len(gws) == 1
	})).Return(&pb.Configuration{
		Gateways: peers,
	})
	if expectedGateways != nil {
		gateways := append(peers, expectedGateways...)
		rc.EXPECT().BuildHelperConfiguration(mock.MatchedBy(func(gws []*pb.Gateway) bool {
			return len(gws) == 2
		})).Return(&pb.Configuration{
			Gateways: gateways,
		})
	}

	deviceAgentListener, stopDeviceAgent := serve(t, NewDeviceAgent(t, ctx, helperListener, rc))

	defer func() {
		stopDeviceAgent()
		stopHelper()
		stopAPIServer()
	}()

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
				t.Logf("receiving agent status: %v", err)
				return
			}
			if err != nil {
				t.Errorf("receiving agent status unexpected error: %v", err)
			} else {
				statusChan <- status
			}
		}
	}()

	lastKnownState := pb.AgentState_Disconnected
	for {
		select {
		case <-ctx.Done():
			t.Errorf("test timed out without agent reaching expected end state: %v, last known state: %v", endState, lastKnownState)
			return
		case status := <-statusChan:
			lastKnownState = status.ConnectionState
			t.Logf("received status: %+v", status.String())
			if status.ConnectionState == endState {
				t.Logf("agent reached expected end state: %v", endState)
				if expectedGateways != nil {
					if len(status.Gateways) == 1 {
						assert.Equal(t, expectedGateways[0].Name, status.Gateways[0].Name)
						return
					}
					continue
				}
				return
			}
		}
	}
}
