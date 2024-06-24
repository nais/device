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
	SetLogger(log logrus.FieldLogger)
}

var _ Notifier = &notifier{}

type notifier struct {
	log logrus.FieldLogger
}

func New(log logrus.FieldLogger) Notifier {
	return &notifier{log: log}
}

func (n *notifier) Infof(format string, args ...any) {
	n.printf(logrus.InfoLevel, format, args...)
}

func (n *notifier) Errorf(format string, args ...any) {
	n.printf(logrus.ErrorLevel, format, args...)
}

func (n *notifier) SetLogger(log logrus.FieldLogger) {
	n.log = log
}

func (n *notifier) printf(logLevel logrus.Level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	logger := n.logFn(logLevel)
	logger(message)
	err := beeep.Notify("naisdevice", message, appIconPath)
	if err != nil {
		n.log.WithError(err).Error("sending notification")
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
