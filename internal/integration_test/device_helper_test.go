package integrationtest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nais/device/internal/helper"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func NewHelper(t *testing.T, log *logrus.Entry, osConfigurator helper.OSConfigurator) *grpc.Server {
	server := grpc.NewServer()
	tempDir, err := os.MkdirTemp("", "naisdevice_helper_test_*")
	assert.NoError(t, err)
	tempfile := filepath.Join(tempDir, "test_interface.conf")
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	helperConfig := helper.Config{
		Interface:           `test_interface`,
		LogLevel:            logrus.DebugLevel.String(),
		WireGuardConfigPath: tempfile,
	}

	deviceHelperServer := helper.NewDeviceHelperServer(log, helperConfig, osConfigurator)
	pb.RegisterDeviceHelperServer(server, deviceHelperServer)

	return server
}
