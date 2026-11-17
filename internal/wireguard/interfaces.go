package wireguard

import (
	"net/netip"

	"github.com/google/gopacket/routing"
)

type NetworkConfigurer interface {
	ApplyWireGuardConfig(peers []Peer) error
	ForwardRoutesV4(routes []string) error
	ForwardRoutesV6(routes []string) error
	SetupInterface() error
	SetupIPTables() error
}

type IPTables interface {
	AppendUnique(table, chain string, rulespec ...string) error
	NewChain(table, chain string) error
	ChangePolicy(table, chain, target string) error
}

// Interface describes the WireGuard interface a NetworkConfigurer manages.
type Interface struct {
	Name       string
	IPv4       *netip.Prefix
	IPv6       *netip.Prefix
	PrivateKey string
	ListenPort int
}

// Forwarding lets tunnel traffic out through the default interfaces, as gateways do.
type Forwarding struct {
	IPTablesV4 IPTables
	IPTablesV6 IPTables
	Router     routing.Router
}
