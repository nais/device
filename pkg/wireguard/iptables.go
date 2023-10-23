package wireguard

import (
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (nc *networkConfigurer) determineDefaultInterfaces() error {
	if len(nc.interfaceIPV4) > 0 && len(nc.wireguardInterface) > 0 {
		return nil
	}

	cmdV4 := exec.Command("ip", "route", "get", "1.1.1.1")
	outV4, err := cmdV4.CombinedOutput()
	if err != nil {
		return err
	}

	nc.defaultInterfaceV4, nc.interfaceIPV4, err = ParseDefaultInterfaceOutputV4(outV4)
	if err != nil {
		return err
	}

	cmdV6 := exec.Command("ip", "route", "get", "2606:4700::1111")
	outV6, err := cmdV6.CombinedOutput()
	if err != nil {
		return err
	}
	nc.defaultInterfaceV6, nc.interfaceIPV6, err = ParseDefaultInterfaceOutputV6(outV6)
	if err != nil {
		return err
	}

	return nil
}

func ParseDefaultInterfaceOutputV4(output []byte) (string, string, error) {
	lines := strings.Split(string(output), "\n")
	parts := strings.Split(lines[0], " ")
	if len(parts) != 9 {
		log.Errorf("wrong number of parts in output: '%v', output: '%v'", len(parts), string(output))
	}

	interfaceName := parts[4]
	if len(interfaceName) < 4 {
		return "", "", fmt.Errorf("weird interface name: '%v'", interfaceName)
	}

	interfaceIP := parts[6]

	if len(strings.Split(interfaceIP, ".")) != 4 {
		return "", "", fmt.Errorf("weird interface ip: '%v'", interfaceIP)
	}

	return interfaceName, interfaceIP, nil
}

func ParseDefaultInterfaceOutputV6(output []byte) (string, string, error) {
	return "", "", nil
}

func (nc *networkConfigurer) SetupIPTables() error {
	err := nc.determineDefaultInterfaces()
	if err != nil {
		return fmt.Errorf("determining default gateway: %w", err)
	}

	err = nc.iptablesV4.ChangePolicy("filter", "FORWARD", "DROP")
	if err != nil {
		return fmt.Errorf("setting FORWARD policy to DROP: %w", err)
	}

	// Allow ESTABLISHED,RELATED from WireGuard to default interface
	err = nc.iptablesV4.AppendUnique("filter", "FORWARD", "-i", nc.wireguardInterface, "-o", nc.defaultInterfaceV4, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default FORWARD outbound-rule: %w", err)
	}

	// Allow ESTABLISHED,RELATED from default interface to WireGuard
	err = nc.iptablesV4.AppendUnique("filter", "FORWARD", "-i", nc.defaultInterfaceV4, "-o", nc.wireguardInterface, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default FORWARD inbound-rule: %w", err)
	}

	// Create and set up LOG_ACCEPT CHAIN
	err = nc.iptablesV4.NewChain("filter", "LOG_ACCEPT")
	if err != nil {
		log.Infof("Creating LOG_ACCEPT chain (probably already exist), error: %v", err)
	}
	err = nc.iptablesV4.AppendUnique("filter", "LOG_ACCEPT", "-j", "LOG", "--log-prefix", "naisdevice-fwd: ", "--log-level", "6")
	if err != nil {
		return fmt.Errorf("adding default LOG_ACCEPT log-rule: %w", err)
	}
	err = nc.iptablesV4.AppendUnique("filter", "LOG_ACCEPT", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default LOG_ACCEPT accept-rule: %w", err)
	}

	return nil
}

func (nc *networkConfigurer) ForwardRoutesV4(routes []string) error {
	var err error

	for _, ip := range routes {
		err = nc.iptablesV4.AppendUnique("nat", "POSTROUTING", "-o", nc.defaultInterfaceV4, "-p", "tcp", "-d", ip, "-j", "SNAT", "--to-source", nc.interfaceIPV4)
		if err != nil {
			return fmt.Errorf("setting up snat: %w", err)
		}

		err = nc.iptablesV4.AppendUnique(
			"filter",
			"FORWARD",
			"--in-interface", nc.wireguardInterface,
			"--out-interface", nc.defaultInterfaceV4,
			"--protocol", "tcp",
			"--syn",
			"--destination", ip,
			"--match", "conntrack",
			"--ctstate", "NEW",
			"--jump", "LOG_ACCEPT",
		)
		if err != nil {
			return fmt.Errorf("setting up iptables log rule: %w", err)
		}
	}

	return nil
}

func (nc *networkConfigurer) ForwardRoutesV6(routes []string) error {
	var err error

	for _, ip := range routes {
		err = nc.iptablesV6.AppendUnique("nat", "POSTROUTING", "-o", nc.defaultInterfaceV4, "-p", "tcp", "-d", ip, "-j", "SNAT", "--to-source", nc.interfaceIPV4)
		if err != nil {
			return fmt.Errorf("setting up snat: %w", err)
		}

		err = nc.iptablesV6.AppendUnique(
			"filter",
			"FORWARD",
			"--in-interface", nc.wireguardInterface,
			"--out-interface", nc.defaultInterfaceV4,
			"--protocol", "tcp",
			"--syn",
			"--destination", ip,
			"--match", "conntrack",
			"--ctstate", "NEW",
			"--jump", "LOG_ACCEPT",
		)
		if err != nil {
			return fmt.Errorf("setting up iptables log rule: %w", err)
		}
	}

	return nil
}
