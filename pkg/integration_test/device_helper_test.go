package integrationtest_test

import (
	"github.com/nais/device/pkg/helper"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"os"
	"path/filepath"
	"testing"
)

func NewHelper(t *testing.T, osConfigurator helper.OSConfigurator) *grpc.Server {
	server := grpc.NewServer()
	tempDir, err := os.MkdirTemp("", "naisdevice_helper_test_*")
	assert.NoError(t, err)
	tempfile := filepath.Join(tempDir, "test_interface.conf")

	deviceHelperServer := helper.DeviceHelperServer{
		Config: helper.Config{
			Interface:           `test_interface`,
			LogLevel:            logrus.DebugLevel.String(),
			WireGuardConfigPath: tempfile,
		},
		OSConfigurator: osConfigurator,
	}
	pb.RegisterDeviceHelperServer(server, &deviceHelperServer)
	return server
}
