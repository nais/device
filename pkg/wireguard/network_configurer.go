package wireguard

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"time"

	"github.com/google/gopacket/routing"
	"github.com/sirupsen/logrus"
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

type subNetworkConfigurer struct {
	ip       *netip.Prefix
	iface    *net.Interface
	src      net.IP
	iptables IPTables
}

type networkConfigurer struct {
	config *Config

	wireguardInterface string
	configPath         string

	v4 *subNetworkConfigurer
	v6 *subNetworkConfigurer

	log *logrus.Entry
}

func detectDefaultRoute(router routing.Router, ip *netip.Prefix) (*net.Interface, net.IP, error) {
	if ip == nil {
		return nil, nil, fmt.Errorf("no IP address configured")
	}

	var testIP net.IP
	if ip.Addr().Is4() {
		testIP = net.ParseIP("1.1.1.1")
	} else {
		testIP = net.ParseIP("2606:4700:4700::1111")
	}

	iface, _, src, err := router.Route(testIP)
	if err != nil {
		return nil, nil, err
	}
	if iface == nil {
		return nil, nil, fmt.Errorf("no default interface found")
	}
	if src == nil {
		return nil, nil, fmt.Errorf("no default source IP found")
	}

	return iface, src, nil
}

func NewConfigurer(log *logrus.Entry, configPath string, ipv4 *netip.Prefix, ipv6 *netip.Prefix, privateKey, wireguardInterface string, listenPort int, iptablesV4, iptablesV6 IPTables, router routing.Router) (NetworkConfigurer, error) {
	nc := &networkConfigurer{
		config: &Config{
			PrivateKey: privateKey,
			ListenPort: listenPort,
		},
		configPath:         configPath,
		wireguardInterface: wireguardInterface,
		v4: &subNetworkConfigurer{
			ip: ipv4,
		},
		v6: &subNetworkConfigurer{
			ip: ipv6,
		},
	}

	if iptablesV4 != nil && router != nil {
		if iface, src, err := detectDefaultRoute(router, ipv4); err != nil {
			return nil, fmt.Errorf("no IPv4 default route found, this is required. err: %w", err)
		} else {
			nc.v4.iface = iface
			nc.v4.src = src
			nc.v4.iptables = iptablesV4
		}
	}
	if iptablesV6 != nil && router != nil {
		if iface, src, err := detectDefaultRoute(router, ipv6); err != nil {
			log.Warnf("no IPv6 default route found, IPv6 will not be configured. err: %v", err)
		} else {
			nc.v6.iface = iface
			nc.v6.src = src
			nc.v6.iptables = iptablesV6
		}
	}

	return nc, nil
}

func (nc *networkConfigurer) SetupInterface() error {
	if nc.v4 == nil && nc.v6 == nil {
		return fmt.Errorf("no IP addresses (v4/v6) configured for interface")
	}

	if err := exec.Command("ip", "link", "del", nc.wireguardInterface).Run(); err != nil {
		nc.log.Infof("pre-deleting WireGuard interface (ok if this fails): %v", err)
	}

	// sysctl net.ipv4.ip_forward
	run := func(commands [][]string) error {
		for _, s := range commands {
			cmd := exec.Command(s[0], s[1:]...)

			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("running %v: %w: %v", cmd, err, string(out))
			} else {
				nc.log.Debugf("%v: %v\n", cmd, string(out))
			}
		}
		return nil
	}

	commands := [][]string{
		{"ip", "link", "add", "dev", nc.wireguardInterface, "type", "wireguard"},
		{"ip", "link", "set", nc.wireguardInterface, "mtu", "1360"},
	}
	if nc.v4.ip != nil {
		commands = append(commands, []string{"ip", "address", "add", "dev", nc.wireguardInterface, nc.v4.ip.String()})
	}
	if nc.v6.ip != nil {
		commands = append(commands, []string{"ip", "address", "add", "dev", nc.wireguardInterface, nc.v6.ip.String()})
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
	nc.log.Debugln(cmd.String())

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sync WireGuard config: %w: out: %v", err, string(out))
	}

	nc.log.Debugf("Actuated WireGuard config at %v", nc.configPath)

	return nil
}
