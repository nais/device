package authenticating

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nais/device/internal/deviceagent/auth"
	"github.com/nais/device/internal/deviceagent/runtimeconfig"
	"github.com/nais/device/internal/deviceagent/statemachine/state"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

		authState := &Authenticating{
			rc: rc,
		}

		assert.Equal(t, state.EventAuthenticated, authState.Enter(ctx).Event)
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

		mockAuth := auth.NewMockHandler(t)
		mockAuth.EXPECT().GetDeviceAgentToken(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tokens, nil)

		authState := &Authenticating{
			authHandler: mockAuth,
			rc:          rc,
		}

		assert.Equal(t, state.EventAuthenticated, authState.Enter(ctx).Event)
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
		notifier.EXPECT().ShowError(expectedError)

		mockAuth := auth.NewMockHandler(t)
		mockAuth.EXPECT().GetDeviceAgentToken(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, expectedError)

		authState := &Authenticating{
			authHandler: mockAuth,
			rc:          rc,
			notifier:    notifier,
		}

		assert.Equal(t, state.EventDisconnect, authState.Enter(ctx).Event)
	})
}
