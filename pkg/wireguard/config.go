package wireguard

import (
	"fmt"
	"io"
	"strings"

	"github.com/nais/device/pkg/ioconvenience"
	"github.com/nais/device/pkg/pb"
)

const PrometheusPeerName = "prometheus"

type Peer interface {
	GetName() string
	GetPublicKey() string
	GetAllowedIPs() []string
	GetEndpoint() string
}

type Config struct {
	Address    string
	ListenPort int
	MTU        int
	Peers      []Peer
	PrivateKey string
}

func (cfg *Config) MarshalINI(w io.Writer) error {
	ew := ioconvenience.NewErrorWriter(w)

	// Global configuration
	fmt.Fprintf(ew, "[Interface]\n")
	fprintNonEmpty(ew, "PrivateKey = %s\n", cfg.PrivateKey)
	if cfg.ListenPort > 0 {
		fmt.Fprintf(ew, "ListenPort = %d\n", cfg.ListenPort)
	}

	// MTU and Address only supported/implemented on Windows platform
	if cfg.MTU > 0 {
		fmt.Fprintf(ew, "MTU = %d\n", cfg.MTU)
	}
	fprintNonEmpty(ew, "Address = %s\n", cfg.Address)

	fmt.Fprintf(ew, "\n")

	for _, peer := range cfg.Peers {
		fmt.Fprintf(ew, "[Peer] # %s\n", peer.GetName())
		fprintNonEmpty(ew, "PublicKey = %s\n", peer.GetPublicKey())
		fprintNonEmpty(ew, "AllowedIPs = %s\n", strings.Join(peer.GetAllowedIPs(), ","))
		fprintNonEmpty(ew, "Endpoint = %s\n", peer.GetEndpoint())
		if peer.GetName() == PrometheusPeerName {
			fmt.Fprint(ew, "PersistentKeepalive = 25\n")
		}
		fmt.Fprintf(ew, "\n")
	}

	_, err := ew.Status()

	return err
}

func fprintNonEmpty(w io.Writer, format string, value string) (int, error) {
	if len(value) == 0 {
		return 0, nil
	}
	return fmt.Fprintf(w, format, value)
}

func MakePeers(devices []*pb.Device, gateways []*pb.Gateway) []Peer {
	peers := make([]Peer, 0, len(devices)+len(gateways))
	for i := range gateways {
		peers = append(peers, gateways[i])
	}
	for i := range devices {
		peers = append(peers, devices[i])
	}
	return peers
}
