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

	last := a.prefix.Addr()
	for _, raw := range taken {
		if raw == "" {
			continue
		}
		addr, err := netip.ParseAddr(raw)
		if err != nil {
			return "", fmt.Errorf("parse allocated IPv6 address %q: %w", raw, err)
		}
		if a.prefix.Contains(addr) && addr.Compare(last) > 0 {
			last = addr
		}
	}

	next := last.Next()
	if !a.prefix.Contains(next) {
		return "", fmt.Errorf("no available IPv6 addresses after %s in prefix %s", last, a.prefix)
	}

	return next.StringExpanded(), nil
}
