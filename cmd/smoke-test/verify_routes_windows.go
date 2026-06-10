package main

import (
	"fmt"
	"log"
	"net"
	"net/netip"

	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

func verifyRoutes(prefixes []netip.Prefix) error {
	log.Println("verifying Windows routes...")

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("lookup interface %q: %w", ifaceName, err)
	}

	ifLUID, err := winipcfg.LUIDFromIndex(uint32(iface.Index))
	if err != nil {
		return fmt.Errorf("get LUID for interface index %d: %w", iface.Index, err)
	}

	for _, prefix := range prefixes {
		nextHop := netip.IPv4Unspecified()
		if prefix.Addr().Is6() {
			nextHop = netip.IPv6Unspecified()
		}
		_, err := ifLUID.Route(prefix, nextHop)
		if err != nil {
			return fmt.Errorf("expected route %s not found on interface %q: %w", prefix, ifaceName, err)
		}
		log.Printf("PASS: route %s found", prefix)
	}

	log.Printf("PASS: all expected routes present on interface %q", ifaceName)
	return nil
}
