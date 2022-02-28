//go:build linux || darwin
// +build linux darwin

package device_agent

import (
	"time"
)

const helperTimeout = 10 * time.Second
