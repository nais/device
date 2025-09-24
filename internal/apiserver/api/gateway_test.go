package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/api"
	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Test_MakeGatewayConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// hash generated with `controlplane-cli passhash --password hunter2`
	mockGateway := &pb.Gateway{
		Endpoint:     "1.2.3.4:56789",
		PublicKey:    "publicKey",
		Name:         "gateway",
		PasswordHash: "$1$5QY7q+KaDZ8EZ+zNaOm2Ag==$BCamA+wMQCcv+QkgJY6H/5Zml5CNq61HkON8tnhUwpj9bq2MkpfPcKLworcMaoVzOfkpEOhf57Btm807pxRAhw==",
		RoutesIPv4: []string{
			"mockroute",
		},
		AccessGroupIDs: []string{"groupId"},
	}
	db := database.NewMockDatabase(t)
	db.On("ReadGateway", mock.Anything, "gateway").Return(mockGateway, nil).Times(2)
	db.EXPECT().GetAcceptances(mock.Anything).Return(map[string]struct{}{"sessionUserId": {}}, nil).Once()

	sessionStore := auth.NewMockSessionStore(t)
	sessionStore.On("All", mock.Anything).Return([]*pb.Session{{
		Device:   &pb.Device{PublicKey: "devicePublicKey"},
		ObjectID: "sessionUserId",
		Expiry:   timestamppb.New(time.Now().Add(24 * time.Hour)),
		Groups:   []string{"groupId"},
	}}, nil)

	gatewayAuthenticator := auth.NewGatewayAuthenticator(db)

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, gatewayAuthenticator, nil, nil, sessionStore, nil)

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

	resp, err := stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, mockGateway.GetRoutesIPv4(), resp.GetRoutesIPv4())

	assert.Len(t, resp.GetDevices(), 1)
	assert.Equal(t, "devicePublicKey", resp.GetDevices()[0].PublicKey)
}
