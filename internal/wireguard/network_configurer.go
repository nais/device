//go:build linux

package wireguard

import (
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/google/gopacket/routing"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const wireguardMTU = 1360

type subNetworkConfigurer struct {
	ip       *netip.Prefix
	iface    *net.Interface
	src      net.IP
	iptables IPTables
}

type networkConfigurer struct {
	privateKey wgtypes.Key
	listenPort int

	wireguardInterface string

	v4 *subNetworkConfigurer
	v6 *subNetworkConfigurer

	log *logrus.Entry
}

func detectDefaultRoute(router routing.Router, ip *netip.Prefix) (*net.Interface, net.IP, error) {
	if ip == nil {
		return nil, nil, fmt.Errorf("no IP address configured")
	}

	var testIP net.IP
	if ip.Addr().Is4() {
		testIP = net.ParseIP("1.1.1.1")
	} else {
		testIP = net.ParseIP("2606:4700:4700::1111")
	}

	iface, _, src, err := router.Route(testIP)
	if err != nil {
		return nil, nil, err
	}
	if iface == nil {
		return nil, nil, fmt.Errorf("no default interface found")
	}
	if src == nil {
		return nil, nil, fmt.Errorf("no default source IP found")
	}

	return iface, src, nil
}

func NewConfigurer(log *logrus.Entry, ipv4 *netip.Prefix, ipv6 *netip.Prefix, privateKey, wireguardInterface string, listenPort int, iptablesV4, iptablesV6 IPTables, router routing.Router) (NetworkConfigurer, error) {
	key, err := wgtypes.ParseKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	nc := &networkConfigurer{
		privateKey:         key,
		listenPort:         listenPort,
		wireguardInterface: wireguardInterface,
		v4: &subNetworkConfigurer{
			ip: ipv4,
		},
		v6: &subNetworkConfigurer{
			ip: ipv6,
		},
		log: log,
	}

	if iptablesV4 != nil && router != nil {
		if iface, src, err := detectDefaultRoute(router, ipv4); err != nil {
			return nil, fmt.Errorf("no IPv4 default route found, this is required. err: %w", err)
		} else {
			nc.v4.iface = iface
			nc.v4.src = src
			nc.v4.iptables = iptablesV4
		}
	}
	if iptablesV6 != nil && router != nil {
		if iface, src, err := detectDefaultRoute(router, ipv6); err != nil {
			log.WithError(err).Warn("no IPv6 default route found, IPv6 will not be configured")
		} else {
			nc.v6.iface = iface
			nc.v6.src = src
			nc.v6.iptables = iptablesV6
		}
	}

	return nc, nil
}

// SetupInterface recreates the WireGuard interface with its addresses, MTU and link state.
func (nc *networkConfigurer) SetupInterface() error {
	addresses := make([]*netlink.Addr, 0, 2)
	for _, prefix := range []*netip.Prefix{nc.v4.ip, nc.v6.ip} {
		if prefix == nil {
			continue
		}
		addr, err := netlink.ParseAddr(prefix.String())
		if err != nil {
			return fmt.Errorf("parse interface address %s: %w", prefix, err)
		}
		addresses = append(addresses, addr)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("no IP addresses (v4/v6) configured for interface")
	}

	if err := nc.deleteInterface(); err != nil {
		return err
	}

	err := netlink.LinkAdd(&netlink.Wireguard{
		Name: nc.wireguardInterface, MTU: wireguardMTU,
	})
	if err != nil {
		return fmt.Errorf("create WireGuard interface %q: %w", nc.wireguardInterface, err)
	}

	if err := nc.configureInterface(addresses); err != nil {
		if delErr := nc.deleteInterface(); delErr != nil {
			return fmt.Errorf("%w (additionally, failed to delete interface: %v)", err, delErr)
		}
		return err
	}

	nc.log.WithField("interface", nc.wireguardInterface).Info("WireGuard interface set up")
	return nil
}

func (nc *networkConfigurer) configureInterface(addresses []*netlink.Addr) error {
	link, err := netlink.LinkByName(nc.wireguardInterface)
	if err != nil {
		return fmt.Errorf("look up WireGuard interface %q: %w", nc.wireguardInterface, err)
	}

	for _, addr := range addresses {
		if err := netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("add address %s to %q: %w", addr, nc.wireguardInterface, err)
		}
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("bring up %q: %w", nc.wireguardInterface, err)
	}

	return nil
}

func (nc *networkConfigurer) deleteInterface() error {
	link, err := netlink.LinkByName(nc.wireguardInterface)
	if errors.As(err, &netlink.LinkNotFoundError{}) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("look up WireGuard interface %q: %w", nc.wireguardInterface, err)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("delete WireGuard interface %q: %w", nc.wireguardInterface, err)
	}
	return nil
}

// ApplyWireGuardConfig configures the interface with exactly the given peers.
// Existing peers are updated in place, so their sessions survive the update.
func (nc *networkConfigurer) ApplyWireGuardConfig(peers []Peer) error {
	client, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("create wgctrl client: %w", err)
	}
	defer func() { _ = client.Close() }()

	device, err := client.Device(nc.wireguardInterface)
	if err != nil {
		return fmt.Errorf("get WireGuard device %q: %w", nc.wireguardInterface, err)
	}

	cfg, err := DeviceConfig(nc.privateKey, &nc.listenPort, peers, device.Peers)
	if err != nil {
		return fmt.Errorf("build WireGuard config: %w", err)
	}

	if err := client.ConfigureDevice(nc.wireguardInterface, *cfg); err != nil {
		return fmt.Errorf("configure WireGuard device %q: %w", nc.wireguardInterface, err)
	}

	nc.log.WithField("peers", len(peers)).Debug("applied WireGuard config")
	return nil
}
