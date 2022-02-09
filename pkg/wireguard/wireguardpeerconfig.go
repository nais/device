package wireguard

import "io"

type WireGuardPeerConfig interface {
	GetTunnelIP() string
	GetWireGuardConfigPath() string
	WriteWireGuardBase(io.Writer) error
}
