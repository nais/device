package notify

import (
	"fmt"

	"github.com/gen2brain/beeep"
	log "github.com/sirupsen/logrus"
)

type logFunc func(string, ...any)

func logfn(logLevel log.Level) logFunc {
	switch logLevel {
	case log.InfoLevel:
		return log.Infof
	case log.ErrorLevel:
		return log.Errorf
	default:
		return log.Printf
	}
}

func Printf(logLevel log.Level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	logger := logfn(logLevel)
	logger(message)
	err := beeep.Notify("naisdevice", message, "../Resources/nais-logo-red.png")
	if err != nil {
		log.Errorf("sending notification: %s", err)
	}
}

func Infof(format string, args ...any) {
	Printf(log.InfoLevel, format, args...)
}

func Errorf(format string, args ...any) {
	Printf(log.ErrorLevel, format, args...)
}
