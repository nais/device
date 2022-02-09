package wireguard

import (
	"os"

	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

type noopConfigurer struct{}

func NewNoOpConfigurer() NetworkConfigurer {
	return &noopConfigurer{}
}

func (n *noopConfigurer) ApplyWireGuardConfig(peers []pb.Peer) error {
	log.Debugf("Applying WireGuard configuration with %d peers", len(peers))
	for _, peer := range peers {
		_ = peer.WritePeerConfig(os.Stdout)
	}
	return nil
}

func (n *noopConfigurer) ForwardRoutes(routes []string) error {
	log.Debugf("Applying %d forwarding routes:", len(routes))
	for i, route := range routes {
		log.Debugf("(%02d) %s", i+1, route)
	}
	return nil
}

func (n *noopConfigurer) ConnectedDeviceCount() (int, error) {
	return 0, nil
}

func (n *noopConfigurer) SetupInterface() error {
	log.Debugf("SetupInterface()")
	return nil
}

func (n *noopConfigurer) SetupIPTables() error {
	log.Debugf("SetupIPTables()")
	return nil
}

var _ NetworkConfigurer = &noopConfigurer{}
