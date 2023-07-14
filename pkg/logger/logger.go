package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

const logfileMaxAge = time.Hour * 24 * 7

func SetupLogger(level, logDir, name string) {
	err := os.MkdirAll(logDir, 0o755)
	if err != nil {
		log.Fatalf("Creating log dir: %v", err)
	}

	err = deleteOldLogFiles(logDir, time.Now().Add(-logfileMaxAge))
	if err != nil {
		log.Errorf("unable to delete old log files: %v", err)
	}

	// clean up old log file without date
	_ = os.Remove(filepath.Join(logDir, name+".log"))

	filename := createLogFileName(name, time.Now())
	logFilePath := filepath.Join(logDir, filename)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o664)
	if err != nil {
		log.Fatalf("unable to open log file %s, error: %v", logFilePath, err)
	}

	// file must be before os.Stdout here because when running as windows service writes to stdout fail.
	mw := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(mw)

	loglevel, err := log.ParseLevel(level)
	if err != nil {
		log.Errorf("unable to parse log level %s, error: %v", level, err)
		return
	}
	log.SetLevel(loglevel)
	log.SetFormatter(&easy.Formatter{TimestampFormat: "2006-01-02 15:04:05.00000", LogFormat: "%time% - [%lvl%] - %msg%\n"})
	log.Infof("Successfully set up logging. Level %s", loglevel)
}

func Setup(level string) {
	log.SetFormatter(&log.JSONFormatter{FieldMap: log.FieldMap{
		log.FieldKeyMsg: "message",
	}})

	l, err := log.ParseLevel(level)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(l)
}
