package main

import (
	"fmt"
	"log"
	"net/netip"

	"github.com/vishvananda/netlink"
)

func verifyRoutes(prefixes []netip.Prefix) error {
	log.Println("verifying Linux routes...")

	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("lookup interface %q: %w", ifaceName, err)
	}

	routes, err := netlink.RouteList(link, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("list routes on %q: %w", ifaceName, err)
	}

	routeSet := make(map[netip.Prefix]bool)
	for _, r := range routes {
		if r.Dst == nil {
			continue
		}
		addr, ok := netip.AddrFromSlice(r.Dst.IP)
		if !ok {
			continue
		}
		ones, _ := r.Dst.Mask.Size()
		routeSet[netip.PrefixFrom(addr.Unmap(), ones)] = true
	}

	for _, want := range prefixes {
		if !routeSet[want] {
			log.Printf("routes on interface %q:", ifaceName)
			for _, r := range routes {
				log.Printf("  %s", r.Dst)
			}
			return fmt.Errorf("expected route %s not found on interface %q", want, ifaceName)
		}
		log.Printf("PASS: route %s found", want)
	}

	log.Printf("PASS: all expected routes present on interface %q", ifaceName)
	return nil
}
