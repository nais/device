package integrationtest_test

import (
	"context"
	"net"
	"sync"
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
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const testGroup = "test-group"

func bufconnDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func serve(t *testing.T, server *grpc.Server, wg *sync.WaitGroup) (*bufconn.Listener, func()) {
	lis := bufconn.Listen(1024 * 1024)
	wg.Add(1)
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("grpc serve error: %v", err)
		}
		wg.Done()
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

type testLogWriter struct {
	t *testing.T
}

func (t *testLogWriter) Write(p []byte) (n int, err error) {
	t.t.Logf("%s", p)
	return len(p), nil
}

func TestIntegration(t *testing.T) {
	t.Parallel()

	logrus.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name             string
		device           *pb.Device
		endState         pb.AgentState
		expectedGateways map[string]*pb.Gateway
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
			expectedGateways: map[string]*pb.Gateway{
				"expected-gateway": {
					Name:           "expected-gateway",
					PublicKey:      "gwPublicKey",
					AccessGroupIDs: []string{testGroup},
					RoutesIPv4:     []string{"1.1.1.1/24", "2.2.2.2/32"},
					RoutesIPv6:     []string{"2001:db8::1/32", "2001:db8::2/32"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logrus.SetOutput(&testLogWriter{t: t})
			wg := &sync.WaitGroup{}
			tableTest(t, wg, test.device, test.endState, test.expectedGateways)
			wg.Wait()
		})
	}
}

func tableTest(t *testing.T, wg *sync.WaitGroup, testDevice *pb.Device, endState pb.AgentState, expectedGateways map[string]*pb.Gateway) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
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

	apiserverPeer := &pb.Gateway{
		Name:      "apiserver",
		PublicKey: "apiserver_public_key",
		Endpoint:  "apiserver_endpoint",
		Ipv4:      "192.0.2.10",
	}

	if expectedGateways == nil {
		expectedGateways = make(map[string]*pb.Gateway)
	}
	// The apiserver is treated as a normal gateway by the device-agent
	expectedGateways[apiserverPeer.Name] = apiserverPeer

	apiserverListener, stopAPIServer := serve(t, NewAPIServer(t, ctx, db), wg)

	osConfigurator := helper.NewMockOSConfigurator(t)
	osConfigurator.EXPECT().SetupInterface(mock.Anything, mock.Anything).Return(nil)
	osConfigurator.EXPECT().SyncConf(mock.Anything, mock.Anything).Return(nil)

	setupRoutesMock := osConfigurator.EXPECT().SetupRoutes(mock.Anything, mock.AnythingOfType("[]*pb.Gateway")).Return(nil)
	if len(expectedGateways) > 1 {
		setupRoutesMock.Run(func(_ context.Context, gateways []*pb.Gateway) {
			for _, gateway := range gateways {
				assert.Equal(t, expectedGateways[gateway.Name].RoutesIPv4, gateway.RoutesIPv4)
				assert.Equal(t, expectedGateways[gateway.Name].RoutesIPv6, gateway.RoutesIPv6)
				assert.Equal(t, expectedGateways[gateway.Name].Endpoint, gateway.Endpoint)
				assert.Equal(t, expectedGateways[gateway.Name].PublicKey, gateway.PublicKey)
				assert.Equal(t, expectedGateways[gateway.Name].AccessGroupIDs, gateway.AccessGroupIDs)
			}
		})
	}

	osConfigurator.EXPECT().TeardownInterface(mock.Anything).Return(nil).Maybe()

	helperListener, stopHelper := serve(t, NewHelper(t, osConfigurator), wg)

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
	rc.EXPECT().APIServerPeer().Return(apiserverPeer)
	rc.EXPECT().BuildHelperConfiguration(mock.MatchedBy(func(gws []*pb.Gateway) bool {
		return len(gws) == 1
	})).Return(&pb.Configuration{
		Gateways: mapValues(expectedGateways),
	})

	deviceAgentListener, stopDeviceAgent := serve(t, NewDeviceAgent(t, wg, ctx, helperListener, rc), wg)

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
	errChan := make(chan error)
	go func() {
		for {
			status, err := statusClient.Recv()
			if grpc_status.Code(err) == codes.Canceled || grpc_status.Code(err) == codes.Unavailable {
				t.Logf("receiving agent status: %v", err)
				return
			}
			if err != nil {
				errChan <- err
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
		case err := <-errChan:
			t.Errorf("receiving agent status unexpected error: %v", err)
		case status := <-statusChan:
			lastKnownState = status.ConnectionState
			t.Logf("received status: %+v", status.String())
			if status.ConnectionState == endState {
				t.Logf("agent reached expected end state: %v", endState)
				for _, gateway := range status.Gateways {
					expectedGateway, exists := expectedGateways[gateway.Name]
					assert.Truef(t, exists, "gateway %s not found in status response", gateway.Name)
					assert.Equal(t, expectedGateway.RoutesIPv4, gateway.RoutesIPv4)
					assert.Equal(t, expectedGateway.RoutesIPv6, gateway.RoutesIPv6)
					assert.Equal(t, expectedGateway.Endpoint, gateway.Endpoint)
					assert.Equal(t, expectedGateway.PublicKey, gateway.PublicKey)
					assert.Equal(t, expectedGateway.AccessGroupIDs, gateway.AccessGroupIDs)
				}

				// test done
				return
			}
		}
	}
}

func mapValues[K comparable, V any](m map[K]V) []V {
	l := make([]V, 0, len(m))
	for _, v := range m {
		l = append(l, v)
	}

	return l
}
