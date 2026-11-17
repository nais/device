//go:build !linux

package wireguard

import "github.com/sirupsen/logrus"

// NewConfigurer returns a configurer that only logs, as host networking is only managed on linux.
func NewConfigurer(log *logrus.Entry, _ Interface, _ *Forwarding) (NetworkConfigurer, error) {
	log.Warn("managing WireGuard on the host is only supported on linux, using no-op configurer")
	return NewNoOpConfigurer(log), nil
}
