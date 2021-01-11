package logger

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func DeviceAgentHelperLogFilePath() string {
	logdir := os.Getenv("PROGRAMDATA") + "\\NAV\\naisdevice"
	if _, err := os.Stat(logdir); err != nil {
		log.Fatalf("Error opening application data folder: %v", err)
	}
	return filepath.Join(logdir, "device-agent-helper.log")
}
