package wireguard

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"

	"github.com/coreos/go-iptables/iptables"
	"github.com/nais/device/pkg/ioconvenience"
	"github.com/nais/device/pkg/pb"

	log "github.com/sirupsen/logrus"
)

type NetworkConfigurer interface {
	ApplyWireGuardConfig(peers []pb.Peer) error
	ForwardRoutes(routes []string) error
	ConnectedDeviceCount() (int, error)
	SetupInterface() error
	SetupIPTables() error
}

type networkConfigurer struct {
	config        WireGuardPeerConfig
	ipTables      *iptables.IPTables
	interfaceName string
	interfaceIP   string
}

func NewConfigurer(config WireGuardPeerConfig, ipTables *iptables.IPTables) NetworkConfigurer {
	return &networkConfigurer{
		config:   config,
		ipTables: ipTables,
	}
}

func (nc *networkConfigurer) SetupInterface() error {
	if err := exec.Command("ip", "link", "del", "wg0").Run(); err != nil {
		log.Infof("pre-deleting WireGuard interface (ok if this fails): %v", err)
	}

	run := func(commands [][]string) error {
		for _, s := range commands {
			cmd := exec.Command(s[0], s[1:]...)

			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("running %v: %w: %v", cmd, err, string(out))
			} else {
				log.Debugf("%v: %v\n", cmd, string(out))
			}
		}
		return nil
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", "wg0", "type", "wireguard"},
		{"ip", "link", "set", "wg0", "mtu", "1360"},
		{"ip", "address", "add", "dev", "wg0", nc.config.GetTunnelIP() + "/21"},
		{"ip", "link", "set", "wg0", "up"},
	}

	return run(commands)
}

// ApplyWireGuardConfig runs syncconfig with the provided WireGuard config
func (nc *networkConfigurer) ApplyWireGuardConfig(peers []pb.Peer) error {
	configFile, err := os.OpenFile(nc.config.GetWireGuardConfigPath(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open WireGuard config file: %w", err)
	}
	defer configFile.Close()

	ew := ioconvenience.NewErrorWriter(configFile)

	_ = nc.config.WriteWireGuardBase(ew)
	_ = WriteWireGuardPeers(ew, peers)

	_, err = ew.Status()
	if err != nil {
		return fmt.Errorf("write wg config: %w", err)
	}

	err = configFile.Close()
	if err != nil {
		return fmt.Errorf("close WireGuard config: %w", err)
	}

	cmd := exec.Command("wg", "syncconf", "wg0", nc.config.GetWireGuardConfigPath())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running syncconf: %w", err)
	}

	log.Debugf("Actuated WireGuard config: %v", nc.config.GetWireGuardConfigPath())

	return nil
}

func (nc *networkConfigurer) ConnectedDeviceCount() (int, error) {
	output, err := exec.Command("wg", "show", "wg0", "endpoints").Output()
	if err != nil {
		return 0, err
	}

	re := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{1,5}`)
	matches := re.FindAll(output, -1)

	return len(matches), nil
}

func WriteWireGuardPeers(w io.Writer, peers []pb.Peer) error {
	ew := ioconvenience.NewErrorWriter(w)
	for _, peer := range peers {
		_ = peer.WritePeerConfig(ew)
	}

	_, err := ew.Status()
	return err
}
