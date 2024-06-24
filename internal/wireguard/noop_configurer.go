package wireguard

import (
	"github.com/sirupsen/logrus"
)

type noopConfigurer struct {
	log *logrus.Entry
}

func NewNoOpConfigurer(log *logrus.Entry) NetworkConfigurer {
	return &noopConfigurer{log: log}
}

func (n *noopConfigurer) ApplyWireGuardConfig(peers []Peer) error {
	n.log.WithField("num_peers", len(peers)).Debug("applying WireGuard configuration")
	for _, peer := range peers {
		n.log.WithField("peer", peer).Debug("log peer")
	}
	return nil
}

func (n *noopConfigurer) ForwardRoutesV4(routes []string) error {
	n.log.WithField("num_routes", len(routes)).Debug("applying forwarding routes")
	for _, route := range routes {
		n.log.WithField("route", route).Debug("log route")
	}
	return nil
}

func (n *noopConfigurer) ForwardRoutesV6(routes []string) error {
	n.log.WithField("num_routes", len(routes)).Debug("applying forwarding routes")
	for _, route := range routes {
		n.log.WithField("route", route).Debug("log route")
	}
	return nil
}

func (n *noopConfigurer) SetupInterface() error {
	n.log.Debug("SetupInterface()")
	return nil
}

func (n *noopConfigurer) SetupIPTables() error {
	n.log.Debug("SetupIPTables()")
	return nil
}

var _ NetworkConfigurer = &noopConfigurer{}
