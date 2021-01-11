package logger

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func DeviceAgentLogFilePath(configDir string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Errorf("opening the user's home directory: %v", err)
		return ""
	}
	return filepath.Join(home, "Library", "Logs", "device-agent.log")
}

func DeviceAgentHelperLogFilePath() string {
	return filepath.Join("/", "Library", "Logs", "device-agent-helper.log")
}
