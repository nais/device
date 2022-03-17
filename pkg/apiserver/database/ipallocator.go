package database

import (
	"fmt"
	"net/netip"
)

type IPAllocator interface {
	NextIP([]string) (string, error)
}

type ipAllocator struct {
	cidr     netip.Prefix
	reserved []string
}

func NewIPAllocator(cidr netip.Prefix, reserved []string) IPAllocator {
	return &ipAllocator{
		cidr:     cidr,
		reserved: reserved,
	}
}

func (i *ipAllocator) NextIP(takenIPs []string) (string, error) {
	takenIPs = append(takenIPs, i.reserved...)
	return findAvailableIP(i.cidr, takenIPs)
}

func findAvailableIP(cidr netip.Prefix, allocated []string) (string, error) {
	allocatedMap := toMap(allocated)
	ips, _ := cidrIPs(cidr)
	for _, ip := range ips {
		if _, found := allocatedMap[ip]; !found {
			return ip, nil
		}
	}
	return "", fmt.Errorf("no available IPs in range %v", cidr)
}

func toMap(strings []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range strings {
		m[s] = struct{}{}
	}
	return m
}

func cidrIPs(cidr netip.Prefix) ([]string, error) {
	var ips []string
	addr := cidr.Addr()
	for ip := addr; cidr.Contains(ip); ip = ip.Next() {
		ips = append(ips, ip.String())
	}

	if cidr.Bits() == 32 {
		return ips, nil
	} else {
		// remove network address and broadcast address
		return ips[1 : len(ips)-1], nil
	}
}
