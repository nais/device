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
	n.log.Debugf("Applying WireGuard configuration with %d peers", len(peers))
	for _, peer := range peers {
		n.log.Debugf("%#v", peer)
	}
	return nil
}

func (n *noopConfigurer) ForwardRoutesV4(routes []string) error {
	n.log.Debugf("Applying %d forwarding routes:", len(routes))
	for i, route := range routes {
		n.log.Debugf("(%02d) %s", i+1, route)
	}
	return nil
}

func (n *noopConfigurer) ForwardRoutesV6(routes []string) error {
	n.log.Debugf("Applying %d forwarding routes:", len(routes))
	for i, route := range routes {
		n.log.Debugf("(%02d) %s", i+1, route)
	}
	return nil
}

func (n *noopConfigurer) SetupInterface() error {
	n.log.Debugf("SetupInterface()")
	return nil
}

func (n *noopConfigurer) SetupIPTables() error {
	n.log.Debugf("SetupIPTables()")
	return nil
}

var _ NetworkConfigurer = &noopConfigurer{}
