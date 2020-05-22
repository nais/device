package logger

import (
	log "github.com/sirupsen/logrus"
)

func Setup(level string, device bool) {
	if device {
		log.SetFormatter(
			&log.TextFormatter{
				FullTimestamp: true,
			})
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}
	l, err := log.ParseLevel(level)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(l)
}
