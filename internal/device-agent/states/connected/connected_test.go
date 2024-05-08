package connected

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nais/device/internal/device-agent/auth"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestConnected_Enter(t *testing.T) {
	logger := logrus.New()

	t.Run("disconnect if unable to configure deviceHelper", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		apiServerPeer := &pb.Gateway{}
		rc.EXPECT().APIServerPeer().Return(apiServerPeer)
		configuration := &pb.Configuration{}
		rc.EXPECT().BuildHelperConfiguration([]*pb.Gateway{
			apiServerPeer,
		}).Return(configuration)

		deviceHelper := pb.NewMockDeviceHelperClient(t)
		deviceHelper.EXPECT().Configure(mock.Anything, configuration).Return(nil, fmt.Errorf("unable to configure"))

		notifier := notify.NewMockNotifier(t)
		notifier.EXPECT().Errorf(mock.Anything, mock.Anything)

		c := &Connected{
			rc:           rc,
			logger:       logger,
			notifier:     notifier,
			deviceHelper: deviceHelper,
		}
		event := c.Enter(ctx)
		assert.Equal(t, statemachine.EventDisconnect, event)
	})

	t.Run("syncConfigLoop", func(t *testing.T) {
		setupMocks := func(t *testing.T) (*runtimeconfig.MockRuntimeConfig, *pb.MockDeviceHelperClient, *notify.MockNotifier) {
			rc := runtimeconfig.NewMockRuntimeConfig(t)
			apiServerPeer := &pb.Gateway{}
			rc.EXPECT().APIServerPeer().Return(apiServerPeer)
			configuration := &pb.Configuration{}
			rc.EXPECT().BuildHelperConfiguration([]*pb.Gateway{
				apiServerPeer,
			}).Return(configuration)

			deviceHelper := pb.NewMockDeviceHelperClient(t)
			deviceHelper.EXPECT().Configure(mock.Anything, configuration).Return(nil, nil)
			deviceHelper.EXPECT().Teardown(mock.Anything, &pb.TeardownRequest{}).Return(nil, nil)

			notifier := notify.NewMockNotifier(t)
			notifier.EXPECT().Errorf(mock.Anything, mock.Anything).Maybe()
			notifier.EXPECT().Infof(mock.Anything, mock.Anything).Maybe()
			return rc, deviceHelper, notifier
		}

		t.Run("returns ErrUnauthenticated", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, notifier := setupMocks(t)

			var token *auth.Tokens
			rc.EXPECT().SetToken(token)

			c := &Connected{
				rc:           rc,
				logger:       logger,
				notifier:     notifier,
				deviceHelper: deviceHelper,
				syncConfigLoop: func(ctx context.Context) error {
					return ErrUnauthenticated
				},
			}
			event := c.Enter(ctx)
			assert.Equal(t, statemachine.EventDisconnect, event)
		})

		t.Run("returns unhandled error", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, notifier := setupMocks(t)

			c := &Connected{
				rc:           rc,
				logger:       logger,
				notifier:     notifier,
				deviceHelper: deviceHelper,
				syncConfigLoop: func(ctx context.Context) error {
					return fmt.Errorf("unhandled error")
				},
			}
			event := c.Enter(ctx)
			assert.Equal(t, statemachine.EventDisconnect, event)
		})

		t.Run("returns context.Canceled", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, notifier := setupMocks(t)

			c := &Connected{
				rc:           rc,
				logger:       logger,
				notifier:     notifier,
				deviceHelper: deviceHelper,
				syncConfigLoop: func(ctx context.Context) error {
					return context.Canceled
				},
			}
			event := c.Enter(ctx)
			assert.Equal(t, statemachine.EventWaitForExternalEvent, event)
		})

		t.Run("returns ErrUnavailable", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, notifier := setupMocks(t)

			alreadyCalled := false

			c := &Connected{
				rc:           rc,
				logger:       logger,
				notifier:     notifier,
				deviceHelper: deviceHelper,
				syncConfigLoop: func(ctx context.Context) error {
					if alreadyCalled {
						return context.Canceled
					}
					alreadyCalled = true
					return ErrUnavailable
				},
			}
			event := c.Enter(ctx)
			assert.Equal(t, statemachine.EventWaitForExternalEvent, event)
		})

		t.Run("returns ErrLostConnection", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, notifier := setupMocks(t)

			alreadyCalled := false

			c := &Connected{
				rc:           rc,
				logger:       logger,
				notifier:     notifier,
				deviceHelper: deviceHelper,
				syncConfigLoop: func(ctx context.Context) error {
					if alreadyCalled {
						return context.Canceled
					}
					alreadyCalled = true
					return ErrLostConnection
				},
			}
			event := c.Enter(ctx)
			assert.Equal(t, statemachine.EventWaitForExternalEvent, event)
		})
	})
}

func TestConnected_defaultSyncConfigLoop(t *testing.T) {
	logger := logrus.New()

	t.Run("unable to get tenant session", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		expectedError := errors.New("expected error")

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().GetTenantSession().Return(nil, expectedError)

		c := &Connected{
			rc:     rc,
			logger: logger,
		}

		err := c.defaultSyncConfigLoop(ctx)
		assert.ErrorIs(t, err, expectedError)
	})

	t.Run("connect to apiserver: unhandled error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		expectedError := errors.New("unhandled error")
		session := &pb.Session{Expiry: timestamppb.New(time.Now().Add(time.Hour)), Key: "key"}

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().GetTenantSession().Return(session, nil)
		rc.EXPECT().ConnectToAPIServer(mock.Anything).Return(nil, func() {}, expectedError)

		c := &Connected{
			rc:     rc,
			logger: logger,
		}

		err := c.defaultSyncConfigLoop(ctx)
		assert.Equal(t, expectedError, err)
	})

	t.Run("connect to apiserver: unavailable error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		unavailableGRPCErr := grpcstatus.Errorf(codes.Unavailable, "unavailable")
		session := &pb.Session{Expiry: timestamppb.New(time.Now().Add(time.Hour)), Key: "key"}

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().GetTenantSession().Return(session, nil)
		rc.EXPECT().ConnectToAPIServer(mock.Anything).Return(nil, func() {}, unavailableGRPCErr)

		c := &Connected{
			rc:     rc,
			logger: logger,
		}

		err := c.defaultSyncConfigLoop(ctx)
		assert.ErrorIs(t, err, ErrUnavailable)
	})

	t.Run("defaultSyncConfigLop.recv", func(t *testing.T) {
		setupMocks := func(ctx context.Context) (*runtimeconfig.MockRuntimeConfig, *pb.MockDeviceHelperClient, *pb.MockAPIServer_GetDeviceConfigurationClient) {
			session := &pb.Session{Expiry: timestamppb.New(time.Now().Add(time.Hour)), Key: "key"}

			getDeviceConfigClient := pb.NewMockAPIServer_GetDeviceConfigurationClient(t)

			apiServerClient := pb.NewMockAPIServerClient(t)
			apiServerClient.EXPECT().GetDeviceConfiguration(mock.Anything, mock.Anything).Return(getDeviceConfigClient, nil)

			rc := runtimeconfig.NewMockRuntimeConfig(t)
			rc.EXPECT().GetTenantSession().Return(session, nil)
			rc.EXPECT().ConnectToAPIServer(mock.Anything).Return(apiServerClient, func() {}, nil)

			deviceHelper := pb.NewMockDeviceHelperClient(t)
			return rc, deviceHelper, getDeviceConfigClient
		}

		t.Run("healthy", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, getDeviceConfigClient := setupMocks(ctx)

			c := &Connected{
				rc:           rc,
				logger:       logger,
				deviceHelper: deviceHelper,
			}
			apiServerPeer := &pb.Gateway{}
			rc.EXPECT().APIServerPeer().Return(apiServerPeer)
			configuration := &pb.Configuration{}
			rc.EXPECT().BuildHelperConfiguration([]*pb.Gateway{apiServerPeer}).Return(configuration)

			deviceHelper.EXPECT().Configure(mock.Anything, configuration).Return(&pb.ConfigureResponse{}, nil)

			alreadyCalled := false
			stopTestErr := errors.New("stop test")
			getDeviceConfigClient.EXPECT().Recv().RunAndReturn(func() (*pb.GetDeviceConfigurationResponse, error) {
				if alreadyCalled {
					return nil, stopTestErr
				}
				alreadyCalled = true
				return &pb.GetDeviceConfigurationResponse{
					Status:   pb.DeviceConfigurationStatus_DeviceHealthy,
					Gateways: []*pb.Gateway{},
				}, nil
			})

			err := c.defaultSyncConfigLoop(ctx)
			assert.ErrorIs(t, err, stopTestErr)
		})
		t.Run("unhealthy", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, getDeviceConfigClient := setupMocks(ctx)
			notifier := notify.NewMockNotifier(t)
			notifier.EXPECT().Errorf(mock.Anything, mock.Anything)

			c := &Connected{
				rc:           rc,
				logger:       logger,
				notifier:     notifier,
				deviceHelper: deviceHelper,
			}

			alreadyCalled := false
			stopTestErr := errors.New("stop test")
			getDeviceConfigClient.EXPECT().Recv().RunAndReturn(func() (*pb.GetDeviceConfigurationResponse, error) {
				if alreadyCalled {
					return nil, stopTestErr
				}
				alreadyCalled = true
				return &pb.GetDeviceConfigurationResponse{
					Status:   pb.DeviceConfigurationStatus_DeviceUnhealthy,
					Gateways: []*pb.Gateway{},
				}, nil
			})

			err := c.defaultSyncConfigLoop(ctx)
			assert.ErrorIs(t, err, stopTestErr)
		})
		t.Run("invalid session", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, getDeviceConfigClient := setupMocks(ctx)

			c := &Connected{
				rc:           rc,
				logger:       logger,
				deviceHelper: deviceHelper,
			}

			getDeviceConfigClient.EXPECT().Recv().Return(&pb.GetDeviceConfigurationResponse{
				Status:   pb.DeviceConfigurationStatus_InvalidSession,
				Gateways: []*pb.Gateway{},
			}, nil)

			err := c.defaultSyncConfigLoop(ctx)
			assert.ErrorIs(t, err, ErrUnauthenticated)
		})

		t.Run("session timeout", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, getDeviceConfigClient := setupMocks(ctx)

			c := &Connected{
				rc:           rc,
				logger:       logger,
				deviceHelper: deviceHelper,
			}

			getDeviceConfigClient.EXPECT().Recv().Return(nil, context.DeadlineExceeded)

			err := c.defaultSyncConfigLoop(ctx)
			assert.ErrorIs(t, err, context.DeadlineExceeded)
		})
		t.Run("err unauthenticated", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, getDeviceConfigClient := setupMocks(ctx)

			c := &Connected{
				rc:           rc,
				logger:       logger,
				deviceHelper: deviceHelper,
			}

			getDeviceConfigClient.EXPECT().Recv().Return(nil, grpcstatus.Errorf(codes.Unavailable, "lost connection"))

			err := c.defaultSyncConfigLoop(ctx)
			assert.ErrorIs(t, err, ErrLostConnection)
		})
		t.Run("unhandled error", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			rc, deviceHelper, getDeviceConfigClient := setupMocks(ctx)

			c := &Connected{
				rc:           rc,
				logger:       logger,
				deviceHelper: deviceHelper,
			}

			expectedError := errors.New("unhandled")
			getDeviceConfigClient.EXPECT().Recv().Return(nil, expectedError)

			err := c.defaultSyncConfigLoop(ctx)
			assert.ErrorIs(t, err, expectedError)
		})
	})

}

func TestConnected_login(t *testing.T) {
	logger := logrus.New()
	t.Run("login: session expired", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		expiredSession := &pb.Session{Expiry: timestamppb.New(time.Now().Add(-time.Hour)), Key: "key"}

		expectedLoginResponse := &pb.APIServerLoginResponse{
			Session: &pb.Session{
				Key:    "newkey",
				Expiry: timestamppb.New(time.Now().Add(time.Hour)),
			},
		}

		apiServerClient := pb.NewMockAPIServerClient(t)
		apiServerClient.EXPECT().Login(mock.Anything, mock.Anything).Return(expectedLoginResponse, nil)

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().GetToken(mock.Anything).Return("token", nil)
		rc.EXPECT().SetTenantSession(expectedLoginResponse.Session).Return(nil)

		deviceHelper := pb.NewMockDeviceHelperClient(t)
		deviceHelper.EXPECT().GetSerial(mock.Anything, mock.Anything).Return(&pb.GetSerialResponse{Serial: "serial"}, nil)

		c := &Connected{
			rc:           rc,
			logger:       logger,
			deviceHelper: deviceHelper,
		}

		session, err := c.login(ctx, apiServerClient, expiredSession)
		assert.NoError(t, err)
		assert.Equal(t, expectedLoginResponse.Session, session)
	})

}
