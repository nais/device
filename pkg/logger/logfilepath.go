//
// +build !darwin
//

package logger

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func DeviceAgentLogFilePath(configDir string) string {
	logPath := filepath.Join(configDir, "logs")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err = os.MkdirAll(logPath, 0755)
		if err != nil {
			log.Fatalf("Unable to create logs directory: %v", err)
		}
	}
	logFilePath := filepath.Join(logPath, "device-agent.log")
	cleanUp(logFilePath)
	return logFilePath
}

// Clean up after the version that messed up by creating a directory inplace of the log file
func cleanUp(logFilePath string) {
	fileInfo, err := os.Stat(logFilePath)
	if err == nil && fileInfo.IsDir() {
		err = os.RemoveAll(logFilePath)
		if err != nil {
			log.Fatalf("Failed to remove directory: %v", err)
		}
	}
}
