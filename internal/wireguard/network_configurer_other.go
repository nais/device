//go:build !linux

package wireguard

import (
	"net/netip"

	"github.com/google/gopacket/routing"
	"github.com/sirupsen/logrus"
)

// NewConfigurer returns a configurer that only logs, as host networking is only managed on linux.
func NewConfigurer(log *logrus.Entry, _ string, _ *netip.Prefix, _ *netip.Prefix, _, _ string, _ int, _, _ IPTables, _ routing.Router) (NetworkConfigurer, error) {
	log.Warn("managing WireGuard on the host is only supported on linux, using no-op configurer")
	return NewNoOpConfigurer(log), nil
}
