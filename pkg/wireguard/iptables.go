package wireguard

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
)

var ErrNetworkUnreachable error = errors.New("network is unreachable")

func (nc *networkConfigurer) SetupIPTables() error {
	err := nc.v4.iptables.ChangePolicy("filter", "FORWARD", "DROP")
	if err != nil {
		return fmt.Errorf("setting FORWARD policy to DROP: %w", err)
	}

	// Allow ESTABLISHED,RELATED from WireGuard to default interface
	err = nc.v4.iptables.AppendUnique("filter", "FORWARD", "-i", nc.wireguardInterface, "-o", nc.v4.iface.Name, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default FORWARD outbound-rule: %w", err)
	}

	// Allow ESTABLISHED,RELATED from default interface to WireGuard
	err = nc.v4.iptables.AppendUnique("filter", "FORWARD", "-i", nc.v4.iface.Name, "-o", nc.wireguardInterface, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default FORWARD inbound-rule: %w", err)
	}

	// Create and set up LOG_ACCEPT CHAIN
	err = nc.v4.iptables.NewChain("filter", "LOG_ACCEPT")
	if err != nil {
		log.Infof("Creating LOG_ACCEPT chain (probably already exist), error: %v", err)
	}
	err = nc.v4.iptables.AppendUnique("filter", "LOG_ACCEPT", "-j", "LOG", "--log-prefix", "naisdevice-fwd: ", "--log-level", "6")
	if err != nil {
		return fmt.Errorf("adding default LOG_ACCEPT log-rule: %w", err)
	}
	err = nc.v4.iptables.AppendUnique("filter", "LOG_ACCEPT", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default LOG_ACCEPT accept-rule: %w", err)
	}

	return nil
}

func (nc *networkConfigurer) ForwardRoutesV4(routes []string) error {
	var err error
	for _, ip := range routes {
		err = nc.v4.iptables.AppendUnique("nat", "POSTROUTING", "-o", nc.v4.iface.Name, "-p", "tcp", "-d", ip, "-j", "SNAT", "--to-source", nc.v4.src.String())
		if err != nil {
			return fmt.Errorf("setting up snat: %w", err)
		}

		err = nc.v4.iptables.AppendUnique(
			"filter",
			"FORWARD",
			"--in-interface", nc.wireguardInterface,
			"--out-interface", nc.v4.iface.Name,
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
		err = nc.v6.iptables.AppendUnique("nat", "POSTROUTING", "-o", nc.v6.iface.Name, "-p", "tcp", "-d", ip, "-j", "SNAT", "--to-source", nc.v6.src.String())
		if err != nil {
			return fmt.Errorf("setting up snat: %w", err)
		}

		err = nc.v6.iptables.AppendUnique(
			"filter",
			"FORWARD",
			"--in-interface", nc.wireguardInterface,
			"--out-interface", nc.v6.iface.Name,
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
