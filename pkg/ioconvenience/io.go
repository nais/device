package ioconvenience

import (
	"io"

	log "github.com/sirupsen/logrus"
)

func CloseReader(r io.ReadCloser) {
	err := r.Close()
	if err != nil {
		log.Warnf("Could not close reader: %s", err)
	}
}
