package logger

import (
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	Agent   = "agent"
	Helper  = "helper"
	Systray = "systray"

	LogDir = "logs"

	logfileMaxAge = time.Hour * 24 * 7
)

func SetupLogger(level, logDir, prefix string) *logrus.Logger {
	log := logrus.New()

	err := os.MkdirAll(logDir, 0o755)
	if err != nil {
		log.WithError(err).Fatal("creating log dir")
	}

	err = deleteOldLogFiles(logDir, time.Now().Add(-logfileMaxAge))
	if err != nil {
		log.WithError(err).Error("unable to delete old log files")
	}

	// clean up old log file without date
	_ = os.Remove(filepath.Join(logDir, prefix+".log"))

	filename := createLogFileName(prefix, time.Now())
	logFilePath := filepath.Join(logDir, filename)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o664)
	if err != nil {
		log.WithError(err).WithField("path", logFilePath).Fatal("unable to open log file")
	}

	// file must be before os.Stdout here because when running as windows service writes to stdout fail.
	mw := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(mw)

	loglevel, err := logrus.ParseLevel(level)
	if err != nil {
		log.WithError(err).WithField("level", level).Error("unable to parse log level")
		return nil
	}
	log.SetLevel(loglevel)
	log.SetFormatter(&logrus.TextFormatter{})
	log.WithField("level", loglevel).Info("successfully set up logging")
	return log
}

func Setup(level string) *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{FieldMap: logrus.FieldMap{
		logrus.FieldKeyMsg: "message",
	}})

	log.SetLevel(logrus.InfoLevel)
	l, err := logrus.ParseLevel(level)
	if err != nil {
		log.WithField("level", level).Warn("parse log level failed, using default level info")
	} else {
		log.SetLevel(l)
	}

	return log
}

func CapturePanic(log logrus.FieldLogger) {
	if err := recover(); err != nil {
		log.WithField("stack", string(debug.Stack())).Errorf("recovered from panic, %T: %x", err, err)
	}
}
