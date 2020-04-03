package database

import (
	"fmt"
	"net"
	"strings"
)

//func main() {
//	ips := []string{"10.255.248.1", "10.255.248.3", "10.255.248.2"}
//	allocated := make(map[string]struct{})
//	for _, allocatedIP := range ips {
//		allocated[allocatedIP] = struct{}{}
//	}
//
//	next, err := FindAvailableIP("10.255.248.0/31", allocated)
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println(next)
//}

func FindAvailableIP(cidr string, allocated map[string]struct{}) (string, error) {
	ips, _ := cidrIPs(cidr)
	for _, ip := range ips {
		fmt.Println(ip)
		if _, found := allocated[ip]; !found {
			return ip, nil
		}
	}
	return "", fmt.Errorf("no available IPs in range %v", cidr)
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
	fmt.Println([]byte(ip))
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
