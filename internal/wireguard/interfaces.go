package wireguard

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
