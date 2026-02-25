package helper

import (
	"context"
	"fmt"
	"net"
	"strings"
	"syscall"

	"github.com/vishvananda/netlink"

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
			if strings.HasPrefix(cidr, TunnelNetworkPrefix) {
				continue
			}

			cidr = strings.TrimSpace(cidr)

			_, dst, err := net.ParseCIDR(cidr)
			if err != nil {
				return routesAdded, fmt.Errorf("parse CIDR %q: %w", cidr, err)
			}

			route := &netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst:       dst,
			}

			if err := netlink.RouteAdd(route); err != nil {
				if err == syscall.EEXIST {
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

	ipv4Addr, err := netlink.ParseAddr(cfg.DeviceIPv4 + "/21")
	if err != nil {
		return fmt.Errorf("parse IPv4 address: %w", err)
	}
	if err := netlink.AddrAdd(link, ipv4Addr); err != nil {
		return fmt.Errorf("add IPv4 address: %w", err)
	}

	ipv6Addr, err := netlink.ParseAddr(cfg.DeviceIPv6 + "/64")
	if err != nil {
		return fmt.Errorf("parse IPv6 address: %w", err)
	}
	if err := netlink.AddrAdd(link, ipv6Addr); err != nil {
		return fmt.Errorf("add IPv6 address: %w", err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("bring interface up: %w", err)
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
