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
	logFilePath := filepath.Join(configDir, "logs", "device-agent.log")
	if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
		err = os.MkdirAll(logFilePath, 0755)
		if err != nil {
			log.Fatalf("Unable to create logs directory: %v", err)
		}
	}
	return logFilePath
}
