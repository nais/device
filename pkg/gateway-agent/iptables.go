package gateway_agent

import (
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (nc *networkConfigurer) determineDefaultInterface() error {
	if len(nc.interfaceIP) > 0 && len(nc.interfaceName) > 0 {
		return nil
	}
	cmd := exec.Command("ip", "route", "get", "1.1.1.1")
	out, err := cmd.CombinedOutput()

	if err != nil {
		return err
	}

	nc.interfaceName, nc.interfaceIP, err = ParseDefaultInterfaceOutput(out)
	return err
}

func ParseDefaultInterfaceOutput(output []byte) (string, string, error) {
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

func (nc *networkConfigurer) SetupIPTables() error {
	err := nc.determineDefaultInterface()
	if err != nil {
		return fmt.Errorf("determining default gateway: %w", err)
	}

	err = nc.ipTables.ChangePolicy("filter", "FORWARD", "DROP")
	if err != nil {
		return fmt.Errorf("setting FORWARD policy to DROP: %w", err)
	}

	// Allow ESTABLISHED,RELATED from wg0 to default interface
	err = nc.ipTables.AppendUnique("filter", "FORWARD", "-i", "wg0", "-o", nc.interfaceName, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default FORWARD outbound-rule: %w", err)
	}

	// Allow ESTABLISHED,RELATED from default interface to wg0
	err = nc.ipTables.AppendUnique("filter", "FORWARD", "-i", nc.interfaceName, "-o", "wg0", "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default FORWARD inbound-rule: %w", err)
	}

	// Create and set up LOG_ACCEPT CHAIN
	err = nc.ipTables.NewChain("filter", "LOG_ACCEPT")
	if err != nil {
		log.Infof("Creating LOG_ACCEPT chain (probably already exist), error: %v", err)
	}
	err = nc.ipTables.AppendUnique("filter", "LOG_ACCEPT", "-j", "LOG", "--log-prefix", "naisdevice-fwd: ", "--log-level", "6")
	if err != nil {
		return fmt.Errorf("adding default LOG_ACCEPT log-rule: %w", err)
	}
	err = nc.ipTables.AppendUnique("filter", "LOG_ACCEPT", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default LOG_ACCEPT accept-rule: %w", err)
	}

	return nil
}

func (nc *networkConfigurer) ForwardRoutes(routes []string) error {
	var err error

	for _, ip := range routes {
		err = nc.ipTables.AppendUnique("nat", "POSTROUTING", "-o", nc.interfaceName, "-p", "tcp", "-d", ip, "-j", "SNAT", "--to-source", nc.interfaceIP)
		if err != nil {
			return fmt.Errorf("setting up snat: %w", err)
		}

		err = nc.ipTables.AppendUnique(
			"filter",
			"FORWARD",
			"--in-interface", "wg0",
			"--out-interface", nc.interfaceName,
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
