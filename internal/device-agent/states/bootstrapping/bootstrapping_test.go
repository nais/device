package bootstrapping

import (
	"context"
	"fmt"
	"github.com/nais/device/internal/device-agent/runtimeconfig"
	"github.com/nais/device/internal/device-agent/statemachine"
	"github.com/nais/device/internal/notify"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestBootstrapping_Enter(t *testing.T) {
	logger := logrus.New()

	t.Run("already enrolled", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().LoadEnrollConfig().Return(nil)

		notifier := notify.NewMockNotifier(t)
		deviceHelper := pb.NewMockDeviceHelperClient(t)

		b := &Bootstrapping{
			rc:           rc,
			logger:       logger,
			notifier:     notifier,
			deviceHelper: deviceHelper,
		}
		event := b.Enter(ctx)
		assert.Equal(t, statemachine.EventBootstrapped, event)
	})

	t.Run("disconnect when fail to get serial", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().LoadEnrollConfig().Return(fmt.Errorf("no enroll config"))

		notifier := notify.NewMockNotifier(t)
		notifier.EXPECT().Errorf(mock.Anything, mock.Anything)

		deviceHelper := pb.NewMockDeviceHelperClient(t)
		deviceHelper.EXPECT().GetSerial(mock.Anything, &pb.GetSerialRequest{}).Return(nil, fmt.Errorf("no serial"))

		b := &Bootstrapping{
			rc:           rc,
			logger:       logger,
			notifier:     notifier,
			deviceHelper: deviceHelper,
		}
		event := b.Enter(ctx)
		assert.Equal(t, statemachine.EventDisconnect, event)
	})

	t.Run("disconnect when unable to enroll", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().LoadEnrollConfig().Return(fmt.Errorf("no enroll config"))
		rc.EXPECT().EnsureEnrolled(mock.Anything, "serial").Return(fmt.Errorf("unable to enroll"))

		notifier := notify.NewMockNotifier(t)
		notifier.EXPECT().Errorf(mock.Anything, mock.Anything)

		deviceHelper := pb.NewMockDeviceHelperClient(t)
		deviceHelper.EXPECT().GetSerial(mock.Anything, &pb.GetSerialRequest{}).Return(&pb.GetSerialResponse{Serial: "serial"}, nil)

		b := &Bootstrapping{
			rc:           rc,
			logger:       logger,
			notifier:     notifier,
			deviceHelper: deviceHelper,
		}
		event := b.Enter(ctx)
		assert.Equal(t, statemachine.EventDisconnect, event)
	})

	t.Run("bootstrapped when enrolled", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		rc := runtimeconfig.NewMockRuntimeConfig(t)
		rc.EXPECT().LoadEnrollConfig().Return(fmt.Errorf("no enroll config"))
		rc.EXPECT().EnsureEnrolled(mock.Anything, "serial").Return(nil)

		notifier := notify.NewMockNotifier(t)

		deviceHelper := pb.NewMockDeviceHelperClient(t)
		deviceHelper.EXPECT().GetSerial(mock.Anything, &pb.GetSerialRequest{}).Return(&pb.GetSerialResponse{Serial: "serial"}, nil)

		b := &Bootstrapping{
			rc:           rc,
			logger:       logger,
			notifier:     notifier,
			deviceHelper: deviceHelper,
		}
		event := b.Enter(ctx)
		assert.Equal(t, statemachine.EventBootstrapped, event)
	})
}
