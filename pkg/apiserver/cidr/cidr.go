package cidr

import (
	"fmt"
	"net"
	"strings"
)

func FindAvailableIP(cidr string, allocated []string) (string, error) {
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

func cidrIPs(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)

	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	if strings.HasSuffix(cidr, "/32") {
		return ips, nil
	} else {
		//remove network address and broadcast address
		return ips[1 : len(ips)-1], nil
	}
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
