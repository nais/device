//go:build darwin || windows

package helper

import (
	"github.com/sirupsen/logrus"
	"golang.zx2c4.com/wireguard/device"
)

func wireguardLogger(log *logrus.Entry) *device.Logger {
	log = log.WithField("component", "wireguard-go")
	return &device.Logger{
		Verbosef: log.Debugf,
		Errorf:   log.Errorf,
	}
}
