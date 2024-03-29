package notify

import (
	"fmt"

	"github.com/gen2brain/beeep"
	"github.com/sirupsen/logrus"
)

type logFunc func(string, ...any)

type Notifier interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}

var _ Notifier = &notifier{}

type notifier struct {
	log *logrus.Entry
}

func New(log *logrus.Entry) Notifier {
	return &notifier{log: log}
}

func (n *notifier) Infof(format string, args ...any) {
	n.printf(logrus.InfoLevel, format, args...)
}

func (n *notifier) Errorf(format string, args ...any) {
	n.printf(logrus.ErrorLevel, format, args...)
}

func (n *notifier) printf(logLevel logrus.Level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	logger := n.logFn(logLevel)
	logger(message)
	err := beeep.Notify("naisdevice", message, appIconPath)
	if err != nil {
		n.log.Errorf("sending notification: %s", err)
	}
}

func (n *notifier) logFn(logLevel logrus.Level) logFunc {
	switch logLevel {
	case logrus.InfoLevel:
		return n.log.Infof
	case logrus.ErrorLevel:
		return n.log.Errorf
	default:
		return n.log.Printf
	}
}
