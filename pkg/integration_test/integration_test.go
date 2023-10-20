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
	"github.com/nais/device/pkg/wireguard"
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

func TestIntegration(t *testing.T) {
	type testCase struct {
		name             string
		device           *pb.Device
		endState         pb.AgentState
		expectedGateways map[string]*pb.Gateway
	}
	tests := []testCase{
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

	logrus.SetLevel(logrus.DebugLevel)
	for _, test := range tests {
		wrap := func(t *testing.T, test testCase) func(*testing.T) {
			return func(t *testing.T) {
				logrus.SetOutput(&testLogWriter{t: t})
				tableTest(t, test.device, test.endState, test.expectedGateways)
			}
		}
		t.Run(test.name, wrap(t, test))
	}
}

func tableTest(t *testing.T, testDevice *pb.Device, endState pb.AgentState, expectedGateways map[string]*pb.Gateway) {
	wg := &sync.WaitGroup{}
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	devicePrivateKey := "devicePrivateKey"

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
	initialPeers := map[string]*pb.Gateway{"apiserver": apiserverPeer}

	if expectedGateways == nil {
		expectedGateways = make(map[string]*pb.Gateway)
	}

	// The apiserver is treated as a normal gateway by the device-agent
	expectedGateways[apiserverPeer.Name] = apiserverPeer

	apiserverListener, stopAPIServer := serve(t, NewAPIServer(t, ctx, db), wg)
	defer stopAPIServer()

	osConfigurator := helper.NewMockOSConfigurator(t)

	configDeviceMatch := func(cfg *pb.Configuration, device *pb.Device) bool {
		return cfg.DeviceIPv4 == device.Ipv4 &&
			cfg.DeviceIPv6 == device.Ipv6 &&
			cfg.PrivateKey == devicePrivateKey
	}

	// expect only the apiserver (once)
	osConfigurator.EXPECT().SetupInterface(mock.AnythingOfType("*context.valueCtx"), mock.MatchedBy(func(cfg *pb.Configuration) bool {
		return configDeviceMatch(cfg, testDevice) &&
			matchExactGateways(initialPeers)(cfg.Gateways)
	})).Return(nil).Once()
	osConfigurator.EXPECT().SyncConf(mock.AnythingOfType("*context.valueCtx"), mock.MatchedBy(func(cfg *pb.Configuration) bool {
		return configDeviceMatch(cfg, testDevice) &&
			matchExactGateways(initialPeers)(cfg.Gateways)
	})).Return(nil)

	if len(expectedGateways) > 1 {
		// expect all gateways
		osConfigurator.EXPECT().SetupInterface(mock.AnythingOfType("*context.valueCtx"), mock.MatchedBy(func(cfg *pb.Configuration) bool {
			return configDeviceMatch(cfg, testDevice) &&
				matchExactGateways(expectedGateways)(cfg.Gateways)
		})).Return(nil)

		osConfigurator.EXPECT().SyncConf(mock.AnythingOfType("*context.valueCtx"), mock.MatchedBy(func(cfg *pb.Configuration) bool {
			return configDeviceMatch(cfg, testDevice) &&
				matchExactGateways(expectedGateways)(cfg.Gateways)
		})).Return(nil)
	}

	setupRoutesMock := osConfigurator.EXPECT().SetupRoutes(mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("[]*pb.Gateway")).Return(nil)
	if len(expectedGateways) > 1 {
		setupRoutesMock.Run(func(_ context.Context, gateways []*pb.Gateway) {
			for _, gateway := range gateways {
				assertEqualGateway(t, expectedGateways[gateway.Name], gateway)
			}
		})
	}

	osConfigurator.EXPECT().TeardownInterface(mock.AnythingOfType("*context.valueCtx")).Return(nil).Maybe()

	helperListener, stopHelper := serve(t, NewHelper(t, osConfigurator), wg)
	defer stopHelper()

	apiDial, err := dial(ctx, apiserverListener)
	assert.NoError(t, err)

	rc := runtimeconfig.NewMockRuntimeConfig(t)
	rc.EXPECT().DialAPIServer(mock.AnythingOfType("*context.timerCtx")).Return(apiDial, nil)
	rc.EXPECT().Tenants().Return([]*pb.Tenant{{
		Name:         "test",
		AuthProvider: pb.AuthProvider_Google,
		Domain:       "test.nais.io",
	}})
	rc.EXPECT().SetToken(mock.AnythingOfType("*auth.Tokens")).Return()
	rc.EXPECT().ResetEnrollConfig().Return()
	rc.EXPECT().GetTenantSession().Return(session, nil)
	rc.EXPECT().LoadEnrollConfig().Return(nil)
	rc.EXPECT().APIServerPeer().Return(apiserverPeer)

	initialHelperConfig := &pb.Configuration{
		Gateways:   mapValues(initialPeers),
		DeviceIPv4: testDevice.Ipv4,
		DeviceIPv6: testDevice.Ipv6,
		PrivateKey: devicePrivateKey,
	}
	rc.EXPECT().BuildHelperConfiguration(mock.MatchedBy(matchExactGateways(initialPeers))).Return(initialHelperConfig)

	if len(expectedGateways) > 1 {
		fullHelperConfig := &pb.Configuration{
			Gateways:   mapValues(expectedGateways),
			DeviceIPv4: testDevice.Ipv4,
			DeviceIPv6: testDevice.Ipv6,
			PrivateKey: devicePrivateKey,
		}
		rc.EXPECT().BuildHelperConfiguration(mock.MatchedBy(matchExactGateways(expectedGateways))).Return(fullHelperConfig)
	}

	for _, gw := range expectedGateways {
		if gw.Name == "apiserver" {
			continue
		}

		var expectedPeers []wireguard.Peer
		expectedPeers = append(expectedPeers, apiserverPeer)

		gatewayNC := wireguard.NewMockNetworkConfigurer(t)
		gatewayNC.EXPECT().ApplyWireGuardConfig(mock.MatchedBy(matchExactPeers(t, expectedPeers))).Return(nil)
		if len(expectedGateways) > 1 {
			expectedPeers = append(expectedPeers, testDevice)
			//
			// gatewayNC.EXPECT().ApplyWireGuardConfig(mock.MatchedBy(matchExactPeers(t, expectedPeers))).Run(func(_ []wireguard.Peer) { gatewayGotDevice <- struct{}{} }).Return(nil)
		}

		gatewayNC.EXPECT().ForwardRoutesV4(gw.GetRoutesIPv4()).Return(nil)
		gatewayNC.EXPECT().ForwardRoutesV6(gw.GetRoutesIPv6()).Return(nil)

		wg.Add(1)
		go func(t *testing.T, gw *pb.Gateway) {
			t.Logf("starting gateway agent %q", gw.GetName())
			err = StartGatewayAgent(t, ctx, gw.GetName(), apiserverListener, apiserverPeer, gatewayNC)

			if grpc_status.Code(err) != codes.Canceled && grpc_status.Code(err) != codes.Unavailable {
				t.Errorf("FAIL: got unexpected error from gateway agent: %v", err)
			}
			wg.Done()
		}(t, gw)
	}

	deviceAgentListener, stopDeviceAgent := serve(t, NewDeviceAgent(t, wg, ctx, helperListener, rc), wg)
	defer stopDeviceAgent()

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
			t.Errorf("FAIL: test timed out without agent reaching expected end state: %v, last known state: %v", endState, lastKnownState)
			return
		case err := <-errChan:
			t.Errorf("FAIL: receiving agent status unexpected error: %v", err)
		case status := <-statusChan:
			lastKnownState = status.ConnectionState
			if status.ConnectionState == endState &&
				matchExactGateways(expectedGateways)(append(status.Gateways, apiserverPeer)) {
				t.Logf("agent reached expected end state: %v", endState)

				// Verify all gateways have the exact expected parameters
				for _, gateway := range status.Gateways {
					expectedGateway, exists := expectedGateways[gateway.Name]
					assert.Truef(t, exists, "gateway %s not found in status response", gateway.Name)
					if !exists {
						continue
					}

					assertEqualGateway(t, expectedGateway, gateway)
				}

				// test done
				return
			} else {
				t.Logf("received non final status: %+v, with gateways: %+v", status.String(), status.Gateways)
			}
		}
	}
}

func assertEqualGateway(t *testing.T, expected, actual *pb.Gateway) {
	t.Helper()
	if expected == nil {
		t.Errorf("FAIL: expected gateway is nil, actual: %+v", actual)
		return
	}
	if actual == nil {
		t.Errorf("FAIL: actual gateway is nil, expected: %+v", expected)
		return
	}
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.RoutesIPv4, actual.RoutesIPv4)
	assert.Equal(t, expected.RoutesIPv6, actual.RoutesIPv6)
	assert.Equal(t, expected.Endpoint, actual.Endpoint)
	assert.Equal(t, expected.PublicKey, actual.PublicKey)
	assert.Equal(t, expected.AccessGroupIDs, actual.AccessGroupIDs)
}

func matchExactGateways(expectedGateways map[string]*pb.Gateway) func([]*pb.Gateway) bool {
	return func(actualGateways []*pb.Gateway) bool {
		if len(actualGateways) != len(expectedGateways) {
			return false
		}

		for _, gateway := range actualGateways {
			if _, exists := expectedGateways[gateway.Name]; exists {
				continue
			} else {
				return false
			}
		}

		for _, expected := range expectedGateways {
			if !gatewayListContains(expected, actualGateways) {
				return false
			}
		}

		return true
	}
}

func matchExactPeers(t *testing.T, expectedPeers []wireguard.Peer) func([]wireguard.Peer) bool {
	return func(peers []wireguard.Peer) bool {
		for _, expectedPeer := range expectedPeers {
			found := false
			for _, peer := range peers {
				if peer.GetName() == expectedPeer.GetName() {
					found = true
				}
			}

			if !found {
				t.Logf("trying to match func: expected peer %s not found in actual peers %+v", expectedPeer.GetName(), peers)
				return false
			}
		}
		return true
	}

}

func gatewayListContains(gatewayToLookFor *pb.Gateway, gateways []*pb.Gateway) bool {
	for _, gateway := range gateways {
		if gatewayToLookFor.Name == gateway.Name {
			return true
		}
	}

	return false
}

func mapValues[K comparable, V any](m map[K]V) []V {
	l := make([]V, 0, len(m))
	for _, v := range m {
		l = append(l, v)
	}

	return l
}

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
