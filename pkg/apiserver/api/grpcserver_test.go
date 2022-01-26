package api_test

import (
	"context"
	"log"
	"net"
	"testing"

	"github.com/nais/device/pkg/apiserver/api"
	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	accessGroups := []string{"auth"}

	apiServer := &database.MockAPIServer{}
	apiServer.On("ReadSessionInfo", mock.Anything, mock.Anything).Return(&pb.Session{Groups: accessGroups}, nil)
	apiServer.On("ReadDeviceById", mock.Anything, mock.Anything).Return(&pb.Device{Healthy: true}, nil)
	apiServer.On("ReadGateways", mock.Anything).Return([]*pb.Gateway{
		{
			Endpoint:       "1.2.3.4:56789",
			PublicKey:      "publicKey",
			Name:           "gateway",
			PasswordHash:   "hunter2",
			AccessGroupIDs: accessGroups,
		},
	}, nil)

	gatewayAuthenticator := auth.NewGatewayAuthenticator(apiServer)

	server := api.NewGRPCServer(apiServer, nil, nil, gatewayAuthenticator, nil)

	pb.RegisterAPIServerServer(s, server)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestGetDeviceConfiguration(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithInsecure(),
	)

	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewAPIServerClient(conn)
	configClient, err := client.GetDeviceConfiguration(ctx, &pb.GetDeviceConfigurationRequest{})
	if err != nil {
		t.Fatalf("ListGateways failed: %v", err)
	}

	resp, err := configClient.Recv()
	if err != nil {
		t.Fatalf("ListGateways().Recv() failed: %v", err)
	}

	gw := resp.Gateways[0]

	assert.Equal(t, "", gw.PasswordHash)
}

func TestGatewayPasswordAuthentication(t *testing.T) {
	apiServer := &database.MockAPIServer{}
	apiServer.On("ReadGateway", mock.Anything, "gateway").Return(&pb.Gateway{
		Endpoint:     "1.2.3.4:56789",
		PublicKey:    "publicKey",
		Name:         "gateway",
		PasswordHash: "hunter2",
	}, nil)
	gatewayAuthenticator := auth.NewGatewayAuthenticator(apiServer)

	server := api.NewGRPCServer(apiServer, nil, nil, gatewayAuthenticator, nil)

	err := server.GetGatewayConfiguration(&pb.GetGatewayConfigurationRequest{Gateway: "gateway", Password: "hunter2"}, nil)
	assert.NoError(t, err)

	err = server.GetGatewayConfiguration(&pb.GetGatewayConfigurationRequest{Gateway: "gateway", Password: "tullepassord"}, nil)
	assert.Equal(t, codes.Unauthenticated, status.Convert(err).Code())
}
