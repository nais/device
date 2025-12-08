package ioconvenience

import (
	"io"

	"github.com/sirupsen/logrus"
)

func CloseWithLog(r io.Closer, log logrus.FieldLogger) {
	err := r.Close()
	if err != nil {
		log.WithError(err).Warn("could not close reader")
	}
}
