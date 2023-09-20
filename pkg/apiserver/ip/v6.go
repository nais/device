package ip

import (
	"fmt"
	"net/netip"
)

type v6Allocator struct {
	prefix *netip.Prefix
}

func NewV6Allocator(prefix *netip.Prefix) Allocator {
	return &v6Allocator{
		prefix: prefix,
	}
}

func (a v6Allocator) NextIP(taken []string) (string, error) {
	if a.prefix == nil {
		return "", fmt.Errorf("no prefix specified in allocator")
	}

	if len(taken) < 1 {
		return a.prefix.Addr().StringExpanded(), nil
	}

	last := taken[len(taken)-1]

	addr, err := netip.ParseAddr(last)
	if err != nil {
		return "", fmt.Errorf("invalid ip address: %s", last)
	}

	next := addr.Next()
	if !a.prefix.Contains(next) {
		return "", fmt.Errorf("ip address %s is not in prefix %s", next, a.prefix)
	}

	return next.StringExpanded(), nil
}
