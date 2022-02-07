package api_test

import (
	"context"
	"net"
	"testing"
	"time"

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
	"google.golang.org/protobuf/types/known/timestamppb"
)

const bufSize = 1024 * 1024

func contextBufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestGetDeviceConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lis := bufconn.Listen(bufSize)

	accessGroups := []string{"auth"}

	db := &database.MockAPIServer{}
	db.On("ReadSessionInfo", mock.Anything, mock.Anything).Return(&pb.Session{Groups: accessGroups}, nil)
	db.On("ReadDeviceById", mock.Anything, mock.Anything).Return(&pb.Device{
		Healthy:        true,
		KolideLastSeen: timestamppb.New(time.Now()),
	}, nil)
	db.On("ReadGateways", mock.Anything).Return([]*pb.Gateway{
		{
			Endpoint:       "1.2.3.4:56789",
			PublicKey:      "publicKey",
			Name:           "gateway",
			PasswordHash:   "hunter2",
			AccessGroupIDs: accessGroups,
		},
	}, nil)

	gatewayAuthenticator := auth.NewGatewayAuthenticator(db)

	server := api.NewGRPCServer(db, nil, nil, gatewayAuthenticator, nil, make(chan struct{}, 10))

	s := grpc.NewServer()
	pb.RegisterAPIServerServer(s, server)
	go s.Serve(lis)

	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(contextBufDialer(lis)),
		grpc.WithInsecure(),
	)
	assert.NoError(t, err)
	defer conn.Close()

	client := pb.NewAPIServerClient(conn)
	configClient, err := client.GetDeviceConfiguration(ctx, &pb.GetDeviceConfigurationRequest{})
	assert.NoError(t, err)

	resp, err := configClient.Recv()
	assert.NoError(t, err)

	gw := resp.Gateways[0]

	assert.Equal(t, "", gw.PasswordHash)

	db.AssertExpectations(t)
}

func TestGatewayPasswordAuthentication(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lis := bufconn.Listen(bufSize)

	// hash generated with `controlplane-cli passhash --password hunter2`
	gwResponse := &pb.Gateway{
		Endpoint:     "1.2.3.4:56789",
		PublicKey:    "publicKey",
		Name:         "gateway",
		PasswordHash: "$1$5QY7q+KaDZ8EZ+zNaOm2Ag==$BCamA+wMQCcv+QkgJY6H/5Zml5CNq61HkON8tnhUwpj9bq2MkpfPcKLworcMaoVzOfkpEOhf57Btm807pxRAhw==",
		Routes: []string{
			"mockroute",
		},
	}
	db := &database.MockAPIServer{}
	db.On("ReadSessionInfos", mock.Anything, mock.Anything).Return([]*pb.Session{}, nil)
	db.On("ReadGateway", mock.Anything, "gateway").Return(gwResponse, nil).Times(2)

	gatewayAuthenticator := auth.NewGatewayAuthenticator(db)

	server := api.NewGRPCServer(db, nil, nil, gatewayAuthenticator, nil, make(chan struct{}, 10))

	s := grpc.NewServer()
	pb.RegisterAPIServerServer(s, server)
	go s.Serve(lis)

	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(contextBufDialer(lis)),
		grpc.WithInsecure(),
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
	assert.Equal(t, gwResponse.Routes, gw.Routes)
}

func TestGatewayPasswordAuthenticationFail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lis := bufconn.Listen(bufSize)

	// hash generated with `controlplane-cli passhash --password hunter2`
	gwResponse := &pb.Gateway{
		Endpoint:     "1.2.3.4:56789",
		PublicKey:    "publicKey",
		Name:         "gateway",
		PasswordHash: "$1$5QY7q+KaDZ8EZ+zNaOm2Ag==$BCamA+wMQCcv+QkgJY6H/5Zml5CNq61HkON8tnhUwpj9bq2MkpfPcKLworcMaoVzOfkpEOhf57Btm807pxRAhw==",
		Routes: []string{
			"mockroute",
		},
	}

	db := &database.MockAPIServer{}
	db.On("ReadGateway", mock.Anything, "gateway").Return(gwResponse, nil).Times(1)

	gatewayAuthenticator := auth.NewGatewayAuthenticator(db)

	server := api.NewGRPCServer(db, nil, nil, gatewayAuthenticator, nil, make(chan struct{}, 10))

	s := grpc.NewServer()
	pb.RegisterAPIServerServer(s, server)
	go s.Serve(lis)

	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(contextBufDialer(lis)),
		grpc.WithInsecure(),
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
