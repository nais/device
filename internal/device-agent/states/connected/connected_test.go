package connected

import (
	"context"
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
	"testing"
	"time"
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
		rc.EXPECT().GetToken(ctx).Return("token", nil)
		rc.EXPECT().SetTenantSession(expectedLoginResponse.Session).Return(nil)

		deviceHelper := pb.NewMockDeviceHelperClient(t)
		deviceHelper.EXPECT().GetSerial(ctx, mock.Anything).Return(&pb.GetSerialResponse{Serial: "serial"}, nil)

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
