package ioconvenience

import (
	"io"

	log "github.com/sirupsen/logrus"
)

func CloseWithLog(r io.Closer) {
	err := r.Close()
	if err != nil {
		log.Warnf("Could not close reader: %s", err)
	}
}
