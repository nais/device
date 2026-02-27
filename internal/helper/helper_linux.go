package helper

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"syscall"

	"github.com/vishvananda/netlink"

	"github.com/nais/device/internal/iputil"
	"github.com/nais/device/internal/wgconfig"
	"github.com/nais/device/pkg/pb"
)

func New(helperConfig Config) *LinuxConfigurator {
	return &LinuxConfigurator{
		helperConfig: helperConfig,
	}
}

type LinuxConfigurator struct {
	helperConfig Config
	tunnelNet    netip.Prefix
}

var _ OSConfigurator = &LinuxConfigurator{}

func (c *LinuxConfigurator) Prerequisites() error {
	return nil
}

func (c *LinuxConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	return wgconfig.ApplyConfig(ctx, c.helperConfig.Interface, cfg)
}

func (c *LinuxConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) (int, error) {
	link, err := netlink.LinkByName(c.helperConfig.Interface)
	if err != nil {
		return 0, fmt.Errorf("lookup interface %q: %w", c.helperConfig.Interface, err)
	}

	routesAdded := 0
	for _, gw := range gateways {
		for _, cidr := range append(gw.GetRoutesIPv4(), gw.GetRoutesIPv6()...) {
			if IsTunnelRoute(c.tunnelNet, cidr) {
				continue
			}

			cidr = strings.TrimSpace(cidr)

			prefix, err := iputil.ParsePrefix(cidr)
			if err != nil {
				return routesAdded, fmt.Errorf("parse route: %w", err)
			}
			addr := prefix.Addr()
			ones := prefix.Bits()
			var bits int
			if addr.Is4() {
				bits = 32
			} else {
				bits = 128
			}
			dst := &net.IPNet{
				IP:   addr.AsSlice(),
				Mask: net.CIDRMask(ones, bits),
			}

			route := &netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst:       dst,
			}

			if err := netlink.RouteAdd(route); err != nil {
				if errors.Is(err, syscall.EEXIST) {
					continue
				}
				return routesAdded, fmt.Errorf("add route %s: %w", cidr, err)
			}
			routesAdded++
		}
	}

	return routesAdded, nil
}

func (c *LinuxConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	tunnelNet, err := TunnelNetworkFromIP(cfg.DeviceIPv4)
	if err != nil {
		return fmt.Errorf("derive tunnel network: %w", err)
	}
	c.tunnelNet = tunnelNet

	if c.interfaceExists(ctx) {
		return nil
	}

	wgLink := &netlink.Wireguard{
		LinkAttrs: netlink.LinkAttrs{
			Name: c.helperConfig.Interface,
			MTU:  wireguardMTU,
		},
	}

	if err := netlink.LinkAdd(wgLink); err != nil {
		return fmt.Errorf("create wireguard interface: %w", err)
	}

	link, err := netlink.LinkByName(c.helperConfig.Interface)
	if err != nil {
		return fmt.Errorf("lookup interface after creation: %w", err)
	}

	// cleanup deletes the interface if any subsequent configuration step fails.
	cleanup := func(cause error) error {
		if delErr := netlink.LinkDel(link); delErr != nil {
			return fmt.Errorf("%w (additionally, failed to delete interface: %v)", cause, delErr)
		}
		return cause
	}

	ipv4Addr, err := netlink.ParseAddr(cfg.DeviceIPv4 + "/21")
	if err != nil {
		return cleanup(fmt.Errorf("parse IPv4 address: %w", err))
	}
	if err := netlink.AddrAdd(link, ipv4Addr); err != nil {
		return cleanup(fmt.Errorf("add IPv4 address: %w", err))
	}

	if cfg.DeviceIPv6 != "" {
		ipv6Addr, err := netlink.ParseAddr(cfg.DeviceIPv6 + "/64")
		if err != nil {
			return cleanup(fmt.Errorf("parse IPv6 address: %w", err))
		}
		if err := netlink.AddrAdd(link, ipv6Addr); err != nil {
			return cleanup(fmt.Errorf("add IPv6 address: %w", err))
		}
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return cleanup(fmt.Errorf("bring interface up: %w", err))
	}

	return nil
}

func (c *LinuxConfigurator) TeardownInterface(ctx context.Context) error {
	if !c.interfaceExists(ctx) {
		return nil
	}

	link, err := netlink.LinkByName(c.helperConfig.Interface)
	if err != nil {
		return fmt.Errorf("lookup interface %q: %w", c.helperConfig.Interface, err)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("delete interface: %w", err)
	}

	return nil
}

func (c *LinuxConfigurator) interfaceExists(ctx context.Context) bool {
	_, err := netlink.LinkByName(c.helperConfig.Interface)
	return err == nil
}
