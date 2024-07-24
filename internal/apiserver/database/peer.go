package database

import (
	"github.com/nais/device/internal/wireguard"
)

type peer struct {
	name, publicKey, ip string
}

var _ wireguard.Peer = &peer{}

// GetAllowedIPs implements wireguard.Peer.
func (p *peer) GetAllowedIPs() []string {
	return []string{p.ip + "/32"}
}

// GetEndpoint implements wireguard.Peer.
func (p *peer) GetEndpoint() string {
	return ""
}

// GetName implements wireguard.Peer.
func (p *peer) GetName() string {
	return p.name
}

// GetPublicKey implements wireguard.Peer.
func (p *peer) GetPublicKey() string {
	return p.publicKey
}
