package wireguard

import (
	"fmt"
	"io"
	"strings"

	"github.com/nais/device/pkg/ioconvenience"
)

type Peer interface {
	GetName() string
	GetPublicKey() string
	GetAllowedIPs() []string
	GetEndpoint() string
}

type Config struct {
	PrivateKey string
	Interface  string
	ListenPort int
	Peers      []Peer
}

func (cfg *Config) MarshalINI(w io.Writer) error {
	ew := ioconvenience.NewErrorWriter(w)

	// Global configuration
	fmt.Fprintf(ew, "[Interface]\n")
	fprintNonEmpty(ew, "PrivateKey = %s\n", cfg.PrivateKey)
	if cfg.ListenPort > 0 {
		fmt.Fprintf(ew, "ListenPort = %d\n", cfg.ListenPort)
	}
	fprintNonEmpty(ew, "Interface = %s\n", cfg.Interface)
	fmt.Fprintf(ew, "\n")

	for _, peer := range cfg.Peers {
		fmt.Fprintf(ew, "[Peer] # %s\n", peer.GetName())
		fprintNonEmpty(ew, "PublicKey = %s\n", peer.GetPublicKey())
		fprintNonEmpty(ew, "AllowedIPs = %s\n", strings.Join(peer.GetAllowedIPs(), ","))
		fprintNonEmpty(ew, "Endpoint = %s\n", peer.GetEndpoint())
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
