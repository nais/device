package helper

import "github.com/sirupsen/logrus"

type Config struct {
	Interface           string
	LogLevel            string
	WireGuardConfigPath string
	log                 *logrus.Entry
}
