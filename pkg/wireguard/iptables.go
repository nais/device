package wireguard

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
)

var ErrNetworkUnreachable error = errors.New("network is unreachable")

func (nc *networkConfigurer) setupIPTables(subconfigurer *subNetworkConfigurer) error {
	err := subconfigurer.iptables.ChangePolicy("filter", "FORWARD", "DROP")
	if err != nil {
		return fmt.Errorf("setting FORWARD policy to DROP: %w", err)
	}

	// Allow ESTABLISHED,RELATED from WireGuard to default interface
	err = subconfigurer.iptables.AppendUnique("filter", "FORWARD", "-i", nc.wireguardInterface, "-o", subconfigurer.iface.Name, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default FORWARD outbound-rule: %w", err)
	}

	// Allow ESTABLISHED,RELATED from default interface to WireGuard
	err = subconfigurer.iptables.AppendUnique("filter", "FORWARD", "-i", subconfigurer.iface.Name, "-o", nc.wireguardInterface, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default FORWARD inbound-rule: %w", err)
	}

	// Create and set up LOG_ACCEPT CHAIN
	err = subconfigurer.iptables.NewChain("filter", "LOG_ACCEPT")
	if err != nil {
		log.Infof("Creating LOG_ACCEPT chain (probably already exist), error: %v", err)
	}
	err = subconfigurer.iptables.AppendUnique("filter", "LOG_ACCEPT", "-j", "LOG", "--log-prefix", "naisdevice-fwd: ", "--log-level", "6")
	if err != nil {
		return fmt.Errorf("adding default LOG_ACCEPT log-rule: %w", err)
	}
	err = subconfigurer.iptables.AppendUnique("filter", "LOG_ACCEPT", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default LOG_ACCEPT accept-rule: %w", err)
	}

	return nil
}

func (nc *networkConfigurer) forwardRoutes(subconfigurer *subNetworkConfigurer, routes []string) error {
	for _, ip := range routes {
		err := subconfigurer.iptables.AppendUnique("nat", "POSTROUTING", "-o", subconfigurer.iface.Name, "-p", "tcp", "-d", ip, "-j", "SNAT", "--to-source", subconfigurer.src.String())
		if err != nil {
			return fmt.Errorf("setting up snat: %w", err)
		}

		err = subconfigurer.iptables.AppendUnique(
			"filter",
			"FORWARD",
			"--in-interface", nc.wireguardInterface,
			"--out-interface", subconfigurer.iface.Name,
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
	return nc.forwardRoutes(nc.v6, routes)
}

func (nc *networkConfigurer) ForwardRoutesV4(routes []string) error {
	return nc.forwardRoutes(nc.v4, routes)
}

func (nc *networkConfigurer) SetupIPTables() error {
	if err := nc.setupIPTables(nc.v4); err != nil {
		return err
	}
	if err := nc.setupIPTables(nc.v6); err != nil {
		return err
	}
	return nil
}
