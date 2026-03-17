package helper

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"

	"github.com/nais/device/internal/iputil"
	"github.com/nais/device/internal/wgconfig"
	"github.com/nais/device/pkg/pb"
)

func New(helperConfig Config, log *logrus.Entry) *LinuxConfigurator {
	return &LinuxConfigurator{
		helperConfig: helperConfig,
		log:          log,
	}
}

type LinuxConfigurator struct {
	helperConfig Config
	mu           sync.RWMutex
	tunnelNet    netip.Prefix
	log          *logrus.Entry
}

var _ OSConfigurator = &LinuxConfigurator{}

func (c *LinuxConfigurator) Prerequisites() error {
	return nil
}

func (c *LinuxConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	for _, gw := range cfg.GetGateways() {
		c.log.WithFields(logrus.Fields{
			"peer":        gw.GetName(),
			"endpoint":    gw.GetEndpoint(),
			"public_key":  fmt.Sprintf("%.8s...", gw.GetPublicKey()),
			"allowed_ips": gw.GetAllowedIPs(),
		}).Debug("configuring wireguard peer")
	}
	err := wgconfig.ApplyConfig(ctx, c.helperConfig.Interface, cfg)
	if err != nil {
		c.log.WithError(err).Debug("wireguard config sync failed")
	} else {
		c.log.Debug("wireguard config sync complete")
	}
	return err
}

func (c *LinuxConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) (int, error) {
	c.mu.RLock()
	tunnelNet := c.tunnelNet
	c.mu.RUnlock()

	link, err := netlink.LinkByName(c.helperConfig.Interface)
	if err != nil {
		return 0, fmt.Errorf("lookup interface %q: %w", c.helperConfig.Interface, err)
	}

	c.log.WithFields(logrus.Fields{
		"interface":    c.helperConfig.Interface,
		"link_index":   link.Attrs().Index,
		"num_gateways": len(gateways),
	}).Debug("setting up routes")

	routesAdded := 0
	for _, gw := range gateways {
		for _, cidr := range append(gw.GetRoutesIPv4(), gw.GetRoutesIPv6()...) {
			if IsTunnelRoute(tunnelNet, cidr) {
				c.log.WithFields(logrus.Fields{
					"cidr":    cidr,
					"gateway": gw.GetName(),
				}).Debug("skipping tunnel route")
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
					c.log.WithFields(logrus.Fields{
						"cidr":    cidr,
						"gateway": gw.GetName(),
					}).Debug("route already exists")
					continue
				}
				return routesAdded, fmt.Errorf("add route %s: %w", cidr, err)
			}
			c.log.WithFields(logrus.Fields{
				"cidr":    cidr,
				"gateway": gw.GetName(),
			}).Debug("route added")
			routesAdded++
		}
	}

	c.log.WithField("routes_added", routesAdded).Debug("route setup complete")
	return routesAdded, nil
}

func (c *LinuxConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	tunnelNet, err := TunnelNetworkFromIP(cfg.DeviceIPv4)
	if err != nil {
		return fmt.Errorf("derive tunnel network: %w", err)
	}
	c.mu.Lock()
	c.tunnelNet = tunnelNet
	c.mu.Unlock()
	c.log.WithFields(logrus.Fields{
		"device_ipv4": cfg.DeviceIPv4,
		"device_ipv6": cfg.DeviceIPv6,
		"tunnel_net":  tunnelNet.String(),
		"interface":   c.helperConfig.Interface,
	}).Debug("setting up interface")

	if c.interfaceExists(ctx) {
		c.log.WithField("interface", c.helperConfig.Interface).Debug("interface already exists, skipping creation")
		return nil
	}

	wgLink := &netlink.Wireguard{
		LinkAttrs: netlink.LinkAttrs{
			Name: c.helperConfig.Interface,
			MTU:  wireguardMTU,
		},
	}

	c.log.WithFields(logrus.Fields{
		"interface": c.helperConfig.Interface,
		"mtu":       wireguardMTU,
	}).Debug("creating wireguard interface")
	if err := netlink.LinkAdd(wgLink); err != nil {
		return fmt.Errorf("create wireguard interface: %w", err)
	}

	link, err := netlink.LinkByName(c.helperConfig.Interface)
	if err != nil {
		return fmt.Errorf("lookup interface after creation: %w", err)
	}
	c.log.WithFields(logrus.Fields{
		"interface": c.helperConfig.Interface,
		"index":     link.Attrs().Index,
		"type":      link.Type(),
	}).Debug("interface created")

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
	c.log.WithField("address", ipv4Addr.String()).Debug("adding IPv4 address")
	if err := netlink.AddrAdd(link, ipv4Addr); err != nil {
		return cleanup(fmt.Errorf("add IPv4 address: %w", err))
	}

	if cfg.DeviceIPv6 != "" {
		ipv6Addr, err := netlink.ParseAddr(cfg.DeviceIPv6 + "/64")
		if err != nil {
			return cleanup(fmt.Errorf("parse IPv6 address: %w", err))
		}
		c.log.WithField("address", ipv6Addr.String()).Debug("adding IPv6 address")
		if err := netlink.AddrAdd(link, ipv6Addr); err != nil {
			return cleanup(fmt.Errorf("add IPv6 address: %w", err))
		}
	}

	c.log.Debug("bringing interface up")
	if err := netlink.LinkSetUp(link); err != nil {
		return cleanup(fmt.Errorf("bring interface up: %w", err))
	}

	c.log.WithField("interface", c.helperConfig.Interface).Debug("interface setup complete")
	return nil
}

func (c *LinuxConfigurator) TeardownInterface(ctx context.Context) error {
	if !c.interfaceExists(ctx) {
		c.log.WithField("interface", c.helperConfig.Interface).Debug("interface does not exist, skipping teardown")
		return nil
	}

	link, err := netlink.LinkByName(c.helperConfig.Interface)
	if err != nil {
		return fmt.Errorf("lookup interface %q: %w", c.helperConfig.Interface, err)
	}

	c.log.WithField("interface", c.helperConfig.Interface).Debug("deleting interface")
	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("delete interface: %w", err)
	}

	c.log.WithField("interface", c.helperConfig.Interface).Debug("interface deleted")
	return nil
}

func (c *LinuxConfigurator) interfaceExists(ctx context.Context) bool {
	_, err := netlink.LinkByName(c.helperConfig.Interface)
	return err == nil
}
