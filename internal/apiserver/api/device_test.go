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

func Test_GetDeviceConfiguration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mockDevice := &pb.Device{
		Id:     123,
		Serial: "deviceSerial",
	}

	mockSession := &pb.Session{
		Key:      "sessionKey",
		Device:   mockDevice,
		ObjectID: "sessionUserId",
		Expiry:   timestamppb.New(time.Now().Add(24 * time.Hour)),
		Groups:   []string{"groupId"},
	}
	db := database.NewMockDatabase(t)
	db.On("ReadDeviceById", mock.Anything, int64(123)).Return(mockDevice, nil).Once()
	db.EXPECT().GetAcceptedAt(mock.Anything, "sessionUserId").Return(nil, nil).Once()

	sessionStore := auth.NewMockSessionStore(t)
	sessionStore.On("Get", mock.Anything, mock.Anything).Return(mockSession, nil).Times(2)

	log := logrus.StandardLogger().WithField("component", "test")
	server := api.NewGRPCServer(ctx, log, db, nil, nil, nil, nil, nil, sessionStore, nil)

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
	stream, err := client.GetDeviceConfiguration(
		ctx,
		&pb.GetDeviceConfigurationRequest{
			SessionKey: mockSession.Key,
		},
	)
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	resp, err := stream.Recv()
	assert.NoError(t, err)
	assert.Len(t, resp.GetIssues(), 1)
	assert.Contains(t, resp.GetIssues()[0].Title, "Do's and don't")
}
