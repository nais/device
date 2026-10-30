// Package iputil provides IP address parsing utilities.
package iputil

import (
	"fmt"
	"net/netip"
	"strings"
)

// ParsePrefix parses s as a CIDR prefix.
// Bare IP addresses without a prefix length are treated as host addresses (/32 or /128).
func ParsePrefix(s string) (netip.Prefix, error) {
	s = strings.TrimSpace(s)

	prefix, err := netip.ParsePrefix(s)
	if err == nil {
		return prefix, nil
	}

	addr, addrErr := netip.ParseAddr(s)
	if addrErr != nil {
		return netip.Prefix{}, fmt.Errorf("parse prefix %q: %w", s, err)
	}

	return netip.PrefixFrom(addr, addr.BitLen()), nil
}
