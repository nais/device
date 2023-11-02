package ioconvenience

import (
	"io"

	"github.com/sirupsen/logrus"
)

func CloseWithLog(log *logrus.Entry, r io.Closer) {
	err := r.Close()
	if err != nil {
		log.Warnf("Could not close reader: %s", err)
	}
}
