package notify

import (
	"fmt"

	"github.com/gen2brain/beeep"
	log "github.com/sirupsen/logrus"
)

type logFunc func(string, ...any)

type Notifier interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

var _ Notifier = &notifier{}

type notifier struct{}

func New() Notifier {
	return &notifier{}
}

func (*notifier) Infof(format string, args ...any) {
	printf(log.InfoLevel, format, args...)
}

func (*notifier) Errorf(format string, args ...any) {
	printf(log.ErrorLevel, format, args...)
}

func printf(logLevel log.Level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	logger := logfn(logLevel)
	logger(message)
	err := beeep.Notify("naisdevice", message, "../Resources/nais-logo-red.png")
	if err != nil {
		log.Errorf("sending notification: %s", err)
	}
}

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
