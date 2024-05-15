package authenticating

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
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAuthenticating(t *testing.T) {
	t.Run("non-expired session", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().GetTenantSession().Return(&pb.Session{
			Key:    "key",
			Expiry: timestamppb.New(time.Now().Add(time.Hour)),
		}, nil)

		state := &Authenticating{
			rc: rc,
		}

		assert.Equal(t, statemachine.EventAuthenticated, state.Enter(ctx).Event)
	})

	t.Run("get token succeeds", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		tokens := &auth.Tokens{IDToken: "id-token"}

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().GetTenantSession().Return(&pb.Session{
			Key:    "key",
			Expiry: timestamppb.New(time.Now().Add(-time.Hour)),
		}, nil)
		rc.EXPECT().GetActiveTenant().Return(&pb.Tenant{AuthProvider: pb.AuthProvider_Google})
		rc.EXPECT().SetToken(tokens)

		state := &Authenticating{
			getToken: func(ctx context.Context, fl logrus.FieldLogger, c oauth2.Config, s string) (*auth.Tokens, error) {
				return tokens, nil
			},
			rc: rc,
		}

		assert.Equal(t, statemachine.EventAuthenticated, state.Enter(ctx).Event)
	})

	t.Run("get token fails", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		expectedError := fmt.Errorf("get token failed")

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().GetTenantSession().Return(&pb.Session{
			Key:    "key",
			Expiry: timestamppb.New(time.Now().Add(-time.Hour)),
		}, nil)
		rc.EXPECT().GetActiveTenant().Return(&pb.Tenant{AuthProvider: pb.AuthProvider_Google})

		notifier := notify.NewMockNotifier(t)
		notifier.EXPECT().Errorf(mock.Anything, expectedError)

		state := &Authenticating{
			getToken: func(ctx context.Context, fl logrus.FieldLogger, c oauth2.Config, s string) (*auth.Tokens, error) {
				return nil, expectedError
			},
			rc:       rc,
			notifier: notifier,
		}

		assert.Equal(t, statemachine.EventDisconnect, state.Enter(ctx).Event)
	})
}
