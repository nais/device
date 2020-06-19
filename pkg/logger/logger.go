package logger

import "C"
import (
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

func SetupDeviceLogger(level, path string) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0664)
	if err != nil {
		log.Fatalf("unable to open log file %s, error: %w", path, err)
	}

	mw := io.MultiWriter(os.Stdout, file)
	log.SetOutput(mw)

	log.Infof("Path: %s", path)
	loglevel, err := log.ParseLevel(level)
	if err != nil {
		log.Errorf("unable to parse log level %s, error: %w", level, err)
		return
	}
	log.SetLevel(loglevel)
	log.SetFormatter(&log.TextFormatter{DisableColors: true, TimestampFormat: "2020-01-02 22:15:22", FullTimestamp: true})
	log.Infof("Successfully set up loging. Level %s", loglevel)
}

func Setup(level string) {
	log.SetFormatter(&log.JSONFormatter{})
	l, err := log.ParseLevel(level)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(l)
}
