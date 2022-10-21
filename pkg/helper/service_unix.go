//go:build linux || darwin

package helper

import "context"

func StartService(programContext context.Context, cancel context.CancelFunc) error {
	return nil
}
