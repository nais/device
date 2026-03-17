package main

import (
	"fmt"
	"log"
	"net"
	"net/netip"
	"syscall"

	"golang.org/x/net/route"
)

func verifyRoutes(prefixes []netip.Prefix) error {
	log.Println("verifying macOS routes...")

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("lookup interface %q: %w", ifaceName, err)
	}

	rib, err := route.FetchRIB(syscall.AF_UNSPEC, syscall.NET_RT_DUMP, 0)
	if err != nil {
		return fmt.Errorf("fetch routing table: %w", err)
	}

	msgs, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return fmt.Errorf("parse routing table: %w", err)
	}

	routeSet := make(map[netip.Prefix]bool)
	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok || rm.Index != iface.Index {
			continue
		}
		p, ok := routeMsgToPrefix(rm)
		if !ok {
			continue
		}
		routeSet[p] = true
	}

	for _, want := range prefixes {
		if !routeSet[want] {
			log.Printf("routes on interface %q (index %d):", ifaceName, iface.Index)
			for p := range routeSet {
				log.Printf("  %s", p)
			}
			return fmt.Errorf("expected route %s not found on interface %q", want, ifaceName)
		}
		log.Printf("PASS: route %s found", want)
	}

	log.Printf("PASS: all expected routes present on interface %q", ifaceName)
	return nil
}

func routeMsgToPrefix(rm *route.RouteMessage) (netip.Prefix, bool) {
	if len(rm.Addrs) <= syscall.RTAX_NETMASK {
		return netip.Prefix{}, false
	}

	dst := rm.Addrs[syscall.RTAX_DST]
	mask := rm.Addrs[syscall.RTAX_NETMASK]

	addr, ok := addrToNetipAddr(dst)
	if !ok {
		return netip.Prefix{}, false
	}

	bits := addr.BitLen()
	if mask != nil {
		bits = maskBits(mask, addr.Is6())
	}

	return netip.PrefixFrom(addr, bits), true
}

func addrToNetipAddr(a route.Addr) (netip.Addr, bool) {
	switch v := a.(type) {
	case *route.Inet4Addr:
		return netip.AddrFrom4(v.IP), true
	case *route.Inet6Addr:
		return netip.AddrFrom16(v.IP), true
	default:
		return netip.Addr{}, false
	}
}

func maskBits(a route.Addr, is6 bool) int {
	switch v := a.(type) {
	case *route.Inet4Addr:
		ones, _ := net.IPv4Mask(v.IP[0], v.IP[1], v.IP[2], v.IP[3]).Size()
		return ones
	case *route.Inet6Addr:
		mask := net.IPMask(v.IP[:])
		ones, _ := mask.Size()
		return ones
	default:
		if is6 {
			return 128
		}
		return 32
	}
}
