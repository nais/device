package ip

import (
	"fmt"
	"net/netip"
)

type v6Allocator struct {
	prefix netip.Prefix
}

func NewV6Allocator(prefix netip.Prefix) Allocator {
	return &v6Allocator{
		prefix: prefix,
	}
}

func (a v6Allocator) NextIP(taken []string) (string, error) {
	if len(taken) < 1 {
		return a.prefix.Addr().StringExpanded(), nil
	}

	last := taken[len(taken)-1]

	addr, err := netip.ParseAddr(last)
	if err != nil {
		return "", fmt.Errorf("invalid ip address: %s", last)
	}

	return addr.Next().StringExpanded(), nil
}
