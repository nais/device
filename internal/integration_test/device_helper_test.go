package integrationtest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nais/device/internal/helper"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func NewHelper(t *testing.T, log *logrus.Entry, osConfigurator helper.OSConfigurator) *grpc.Server {
	server := grpc.NewServer()
	testDir := filepath.Join(os.TempDir(), "naisdevice-tests")
	err := os.MkdirAll(testDir, 0o755)
	if err != nil {
		t.Fatalf("unable to setup test dir for helper tests: %v", err)
	}
	tempDir, err := os.MkdirTemp(testDir, "naisdevice_helper_test_*")
	assert.NoError(t, err)
	tempfile := filepath.Join(tempDir, "test_interface.conf")
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("test failed, leaving temp dir in: %v", tempDir)
		} else {
			err := os.RemoveAll(tempDir)
			if err != nil {
				t.Logf("unable to clean up temp dir: %v", err)
			} else {
				t.Logf("cleaned up temp dir: %v", tempDir)
			}
		}
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
