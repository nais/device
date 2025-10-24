package api_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/api"
	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/ioconvenience"
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
	type input struct {
		kolideEnabled bool
		termsAccepted bool
		inGroup       bool
	}
	type output struct {
		getsConfig bool
		issues     []string // substring of tite
	}
	tests := []struct {
		in  input
		out output
	}{
		{
			in: input{
				kolideEnabled: true,
				termsAccepted: true,
				inGroup:       true,
			},
			out: output{
				getsConfig: true,
				issues:     nil,
			},
		},
		{
			in: input{
				kolideEnabled: true,
				termsAccepted: true,
				inGroup:       false,
			},
			out: output{
				getsConfig: false,
				issues:     nil,
			},
		},
		{
			in: input{
				kolideEnabled: true,
				termsAccepted: false,
				inGroup:       true,
			},
			out: output{
				getsConfig: false,
				issues:     []string{"Do's and don'ts"},
			},
		},
		{
			in: input{
				kolideEnabled: true,
				termsAccepted: false,
				inGroup:       false,
			},
			out: output{
				getsConfig: false,
				issues:     []string{"Do's and don'ts"},
			},
		},
		{
			in: input{
				kolideEnabled: false,
				termsAccepted: true,
				inGroup:       true,
			},
			out: output{
				getsConfig: true,
				issues:     nil,
			},
		},
		{
			in: input{
				kolideEnabled: false,
				termsAccepted: true,
				inGroup:       false,
			},
			out: output{
				getsConfig: false,
				issues:     nil,
			},
		},
		{
			in: input{
				kolideEnabled: false,
				termsAccepted: false,
				inGroup:       true,
			},
			out: output{
				getsConfig: true,
				issues:     nil,
			},
		},
		{
			in: input{
				kolideEnabled: false,
				termsAccepted: false,
				inGroup:       false,
			},
			out: output{
				getsConfig: false,
				issues:     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("kolide=%v, terms=%v, group=%v", tt.in.kolideEnabled, tt.in.termsAccepted, tt.in.inGroup), func(t *testing.T) {
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
			}
			if tt.in.inGroup {
				mockSession.Groups = []string{"groupId"}
			}

			sessionStore := auth.NewMockSessionStore(t)
			sessionStore.EXPECT().Get(mock.Anything, mock.Anything).Return(mockSession, nil).Times(2)

			mockGateway := &pb.Gateway{
				Name:           "gateway1",
				AccessGroupIDs: []string{"groupId"},
			}

			db := database.NewMockDatabase(t)
			db.EXPECT().ReadDeviceByID(mock.Anything, int64(123)).Return(mockDevice, nil).Once()
			if tt.in.kolideEnabled {
				if tt.in.termsAccepted {
					db.EXPECT().GetAcceptedAt(mock.Anything, "sessionUserId").Return(timestamppb.Now(), nil).Once()
				} else {
					db.EXPECT().GetAcceptedAt(mock.Anything, "sessionUserId").Return(nil, nil).Once()
				}
			}
			db.EXPECT().ReadGateways(mock.Anything).Return([]*pb.Gateway{mockGateway}, nil).Maybe()

			log := logrus.StandardLogger().WithField("component", "test")
			server := api.NewGRPCServer(ctx, log, db, nil, nil, nil, nil, sessionStore, nil, tt.in.kolideEnabled)

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
			defer ioconvenience.CloseWithLog(log, conn)

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

			for _, expected := range tt.out.issues {
				found := false
				for _, issue := range resp.GetIssues() {
					if strings.Contains(issue.GetTitle(), expected) {
						found = true
						break
					}
				}
				assert.Truef(t, found, "expected issue not found: %q in %+v", expected, resp.GetIssues())
			}

			if tt.out.getsConfig {
				gateways := resp.GetGateways()
				found := false
				for _, gateway := range gateways {
					if mockGateway.Name == gateway.GetName() {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected gateway not found: %q in %+v", mockGateway.Name, gateways)
				}
			}
		})
	}
}
