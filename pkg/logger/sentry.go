package logger

import (
	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
)

type sentryHook struct {
	logLevelsToCapture map[log.Level]sentry.Level
}

func NewSentryHook() *sentryHook {
	return &sentryHook{
		logLevelsToCapture: map[log.Level]sentry.Level{
			log.ErrorLevel: sentry.LevelError,
			log.WarnLevel:  sentry.LevelWarning,
		},
	}
}

func (s sentryHook) Levels() []log.Level {
	var levels []log.Level
	for k := range s.logLevelsToCapture {
		levels = append(levels, k)
	}

	return levels
}

func (s sentryHook) Fire(entry *log.Entry) error {
	event := sentry.NewEvent()
	event.Message = entry.Message
	event.Level = s.logLevelsToCapture[entry.Level]
	sentry.CaptureEvent(event)
	return nil
}
