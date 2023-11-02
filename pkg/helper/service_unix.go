//go:build linux || darwin

package helper

import (
	"context"

	"github.com/sirupsen/logrus"
)

func StartService(_ *logrus.Entry, programContext context.Context, cancel context.CancelFunc) error {
	return nil
}
