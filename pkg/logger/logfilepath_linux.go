package logger

import (
	"path/filepath"
)

func DeviceAgentHelperLogFilePath() string {
	return filepath.Join("/", "var", "log", "device-agent-helper.log")
}
