package api_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/api"
	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const bufSize = 1024 * 1024

var testDevice = &pb.Device{
	Serial:   "serial",
	Platform: "darwin",
	Username: "user@example.com",
}

func contextBufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestGetDeviceConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	accessGroups := []string{"auth"}

	db := database.NewMockDatabase(t)
	db.On("ReadSessionInfo", mock.Anything, mock.Anything).Return(
		&pb.Session{
			Groups: accessGroups,
			Expiry: timestamppb.New(time.Now().Add(10 * time.Second)),
			Device: testDevice,
		}, nil)
	db.EXPECT().GetAcceptedAt(mock.Anything, mock.Anything).Return(timestamppb.Now(), nil)
	db.On("ReadDeviceById", mock.Anything, mock.Anything).Return(testDevice, nil)
	db.On("ReadGateways", mock.Anything).Return([]*pb.Gateway{
		{
			Endpoint:       "1.2.3.4:56789",
			PublicKey:      "publicKey",
			Name:           "gateway",
			PasswordHash:   "hunter2",
			AccessGroupIDs: accessGroups,
		},
		{
			Endpoint:       "1.2.3.5:56789",
			PublicKey:      "publicKey2",
			Name:           "gateway2",
			PasswordHash:   "hunter3",
			AccessGroupIDs: []string{"not authorized"},
		},
	}, nil)

	kolideClient := &kolide.FakeClient{}

	gatewayAuthenticator := auth.NewGatewayAuthenticator(db)

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, gatewayAuthenticator, nil, auth.NewSessionStore(db), kolideClient, true)

	s := grpc.NewServer()
	pb.RegisterAPIServerServer(s, server)
	lis := bufconn.Listen(bufSize)
	go func() {
		err := s.Serve(lis)
		assert.NoError(t, err)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(contextBufDialer(lis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(t, err)
	defer conn.Close()

	client := pb.NewAPIServerClient(conn)
	configClient, err := client.GetDeviceConfiguration(ctx, &pb.GetDeviceConfigurationRequest{})
	assert.NoError(t, err)

	resp, err := configClient.Recv()
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Gateways)

	gw := resp.Gateways[0]

	assert.Len(t, resp.Gateways, 1)
	assert.Equal(t, "", gw.PasswordHash)
	assert.Equal(t, "gateway", gw.Name)
}

func TestGatewayPasswordAuthentication(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// hash generated with `controlplane-cli passhash --password hunter2`
	gwResponse := &pb.Gateway{
		Endpoint:     "1.2.3.4:56789",
		PublicKey:    "publicKey",
		Name:         "gateway",
		PasswordHash: "$1$5QY7q+KaDZ8EZ+zNaOm2Ag==$BCamA+wMQCcv+QkgJY6H/5Zml5CNq61HkON8tnhUwpj9bq2MkpfPcKLworcMaoVzOfkpEOhf57Btm807pxRAhw==",
		RoutesIPv4: []string{
			"mockroute",
		},
	}
	db := database.NewMockDatabase(t)
	db.On("ReadGateway", mock.Anything, "gateway").Return(gwResponse, nil).Times(2)
	db.EXPECT().GetAcceptances(mock.Anything).Return(map[string]struct{}{}, nil).Once()

	sessionStore := auth.NewMockSessionStore(t)
	sessionStore.On("All", mock.Anything).Return([]*pb.Session{}, nil)

	gatewayAuthenticator := auth.NewGatewayAuthenticator(db)

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, gatewayAuthenticator, nil, sessionStore, nil, false)

	s := grpc.NewServer()
	pb.RegisterAPIServerServer(s, server)
	lis := bufconn.Listen(bufSize)
	go func() {
		err := s.Serve(lis)
		assert.NoError(t, err)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(contextBufDialer(lis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(t, err)
	defer conn.Close()

	client := pb.NewAPIServerClient(conn)

	// Test authenticated call with correct password
	stream, err := client.GetGatewayConfiguration(
		ctx,
		&pb.GetGatewayConfigurationRequest{
			Gateway:  "gateway",
			Password: "hunter2",
		},
	)
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	gw, err := stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, gwResponse.GetRoutesIPv4(), gw.GetRoutesIPv4())
}

func TestGatewayPasswordAuthenticationFail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// hash generated with `controlplane-cli passhash --password hunter2`
	gwResponse := &pb.Gateway{
		Endpoint:     "1.2.3.4:56789",
		PublicKey:    "publicKey",
		Name:         "gateway",
		PasswordHash: "$1$5QY7q+KaDZ8EZ+zNaOm2Ag==$BCamA+wMQCcv+QkgJY6H/5Zml5CNq61HkON8tnhUwpj9bq2MkpfPcKLworcMaoVzOfkpEOhf57Btm807pxRAhw==",
		RoutesIPv4: []string{
			"mockroute",
		},
	}

	db := database.NewMockDatabase(t)
	db.On("ReadGateway", mock.Anything, "gateway").Return(gwResponse, nil).Times(1)

	gatewayAuthenticator := auth.NewGatewayAuthenticator(db)

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, gatewayAuthenticator, nil, nil, nil, false)

	s := grpc.NewServer()
	pb.RegisterAPIServerServer(s, server)
	lis := bufconn.Listen(bufSize)
	go func() {
		err := s.Serve(lis)
		assert.NoError(t, err)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(contextBufDialer(lis)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(t, err)
	defer conn.Close()

	client := pb.NewAPIServerClient(conn)

	// Test authenticated call with correct password
	stream, err := client.GetGatewayConfiguration(
		ctx,
		&pb.GetGatewayConfigurationRequest{
			Gateway:  "gateway",
			Password: "wrong-password",
		},
	)
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	gw, err := stream.Recv()
	assert.Nil(t, gw)
	assert.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}
