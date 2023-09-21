package wireguard

import (
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"
)

type NetworkConfigurer interface {
	ApplyWireGuardConfig(peers []Peer) error
	ForwardRoutes(routes []string) error
	SetupInterface() error
	SetupIPTables() error
}

type IPTables interface {
	AppendUnique(table, chain string, rulespec ...string) error
	NewChain(table, chain string) error
	ChangePolicy(table, chain, target string) error
}

type networkConfigurer struct {
	config             *Config
	ipTables           IPTables
	wireguardInterface string
	defaultInterface   string
	interfaceIP        string
	configPath         string
	ipv4               *netip.Prefix
	ipv6               *netip.Prefix
}

func NewConfigurer(configPath string, ipv4 *netip.Prefix, ipv6 *netip.Prefix, privateKey, wireguardInterface string, listenPort int, ipTables IPTables) NetworkConfigurer {
	return &networkConfigurer{
		config: &Config{
			PrivateKey: privateKey,
			ListenPort: listenPort,
		},
		configPath:         configPath,
		wireguardInterface: wireguardInterface,
		ipTables:           ipTables,
		ipv4:               ipv4,
		ipv6:               ipv6,
	}
}

func (nc *networkConfigurer) SetupInterface() error {
	if nc.ipv4 == nil && nc.ipv6 == nil {
		return fmt.Errorf("no IP addresses (v4/v6) configured for interface")
	}

	if err := exec.Command("ip", "link", "del", nc.wireguardInterface).Run(); err != nil {
		log.Infof("pre-deleting WireGuard interface (ok if this fails): %v", err)
	}

	// sysctl net.ipv4.ip_forward

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
		{"ip", "link", "add", "dev", nc.wireguardInterface, "type", "wireguard"},
		{"ip", "link", "set", nc.wireguardInterface, "mtu", "1360"},
	}
	if nc.ipv4 != nil {
		commands = append(commands, []string{"ip", "address", "add", "dev", nc.wireguardInterface, nc.ipv4.String()})
	}
	if nc.ipv6 != nil {
		commands = append(commands, []string{"ip", "address", "add", "dev", nc.wireguardInterface, nc.ipv6.String()})
	}
	commands = append(commands, []string{"ip", "link", "set", nc.wireguardInterface, "up"})

	return run(commands)
}

// ApplyWireGuardConfig runs syncconfig with the provided WireGuard config
func (nc *networkConfigurer) ApplyWireGuardConfig(peers []Peer) error {
	configFile, err := os.OpenFile(nc.configPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open WireGuard config file: %w", err)
	}
	defer configFile.Close()

	nc.config.Peers = peers
	err = nc.config.MarshalINI(configFile)
	if err != nil {
		return fmt.Errorf("write WireGuard config: %w", err)
	}

	// err = configFile.Sync()
	// if err != nil {
	// 	return fmt.Errorf("make sure contents are written to disk: %w", err)
	// }

	err = configFile.Close()
	if err != nil {
		return fmt.Errorf("close WireGuard config: %w", err)
	}

	time.Sleep(1 * time.Second) // TODO: switch to configFile.Sync() commented out above
	cmd := exec.Command("wg", "syncconf", nc.wireguardInterface, nc.configPath)
	log.Debugln(cmd.String())

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sync WireGuard config: %w: out: %v", err, string(out))
	}

	log.Debugf("Actuated WireGuard config at %v", nc.configPath)

	return nil
}

func (nc *networkConfigurer) ConnectedDeviceCount() (int, error) {
	output, err := exec.Command("wg", "show", nc.wireguardInterface, "endpoints").Output()
	if err != nil {
		return 0, err
	}

	re := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}:\d{1,5}`)
	matches := re.FindAll(output, -1)

	return len(matches), nil
}
