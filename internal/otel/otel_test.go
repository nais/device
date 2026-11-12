package otel_test

import (
	"context"
	"testing"

	"github.com/nais/device/internal/otel"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupOTelSDK_Disabled(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "true")

	shutdown, err := otel.SetupOTelSDK(context.Background(), "test", logrus.New())
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))

	_, span := otel.Start(context.Background(), "span")
	assert.False(t, span.IsRecording())
}
