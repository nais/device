package gateway_agent

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func SetupIptables(cfg Config) error {
	err := cfg.IPTables.ChangePolicy("filter", "FORWARD", "DROP")
	if err != nil {
		return fmt.Errorf("setting FORWARD policy to DROP: %w", err)
	}

	// Allow ESTABLISHED,RELATED from wg0 to default interface
	err = cfg.IPTables.AppendUnique("filter", "FORWARD", "-i", "wg0", "-o", cfg.DefaultInterface, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default FORWARD outbound-rule: %w", err)
	}

	// Allow ESTABLISHED,RELATED from default interface to wg0
	err = cfg.IPTables.AppendUnique("filter", "FORWARD", "-i", cfg.DefaultInterface, "-o", "wg0", "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default FORWARD inbound-rule: %w", err)
	}

	// Create and set up LOG_ACCEPT CHAIN
	err = cfg.IPTables.NewChain("filter", "LOG_ACCEPT")
	if err != nil {
		log.Infof("Creating LOG_ACCEPT chain (probably already exist), error: %v", err)
	}
	err = cfg.IPTables.AppendUnique("filter", "LOG_ACCEPT", "-j", "LOG", "--log-prefix", "naisdevice-fwd: ", "--log-level", "6")
	if err != nil {
		return fmt.Errorf("adding default LOG_ACCEPT log-rule: %w", err)
	}
	err = cfg.IPTables.AppendUnique("filter", "LOG_ACCEPT", "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("adding default LOG_ACCEPT accept-rule: %w", err)
	}

	return nil
}

func (nc *networkConfigurer) ForwardRoutes(routes []string) error {
	var err error

	for _, ip := range routes {
		err = nc.config.IPTables.AppendUnique("nat", "POSTROUTING", "-o", nc.config.DefaultInterface, "-p", "tcp", "-d", ip, "-j", "SNAT", "--to-source", nc.config.DefaultInterfaceIP)
		if err != nil {
			return fmt.Errorf("setting up snat: %w", err)
		}

		err = nc.config.IPTables.AppendUnique(
			"filter",
			"FORWARD",
			"--in-interface", "wg0",
			"--out-interface", nc.config.DefaultInterface,
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
