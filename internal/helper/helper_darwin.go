package helper

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"

	"github.com/nais/device/internal/wgconfig"
	"github.com/nais/device/pkg/pb"
)

type DarwinConfigurator struct {
	helperConfig Config

	mu        sync.Mutex
	wgDevice  *device.Device
	tunDev    tun.Device
	uapi      net.Listener
	ifaceName string // actual interface name assigned by macOS (may differ from requested)
}

var _ OSConfigurator = &DarwinConfigurator{}

func New(helperConfig Config) *DarwinConfigurator {
	return &DarwinConfigurator{
		helperConfig: helperConfig,
	}
}

func (c *DarwinConfigurator) Prerequisites() error {
	return nil
}

func (c *DarwinConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	c.mu.Lock()
	ifaceName := c.ifaceName
	c.mu.Unlock()
	if ifaceName == "" {
		ifaceName = c.helperConfig.Interface
	}
	return wgconfig.ApplyConfig(ctx, ifaceName, cfg)
}

func (c *DarwinConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) (int, error) {
	c.mu.Lock()
	ifaceName := c.ifaceName
	c.mu.Unlock()
	if ifaceName == "" {
		ifaceName = c.helperConfig.Interface
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return 0, fmt.Errorf("lookup interface %q: %w", ifaceName, err)
	}

	fd, err := unix.Socket(unix.AF_ROUTE, unix.SOCK_RAW, unix.AF_UNSPEC)
	if err != nil {
		return 0, fmt.Errorf("open routing socket: %w", err)
	}
	defer func() { _ = unix.Close(fd) }()

	routesAdded := 0
	for _, gw := range gateways {
		for _, cidr := range gw.GetRoutesIPv4() {
			if strings.HasPrefix(cidr, TunnelNetworkPrefix) {
				continue
			}
			if err := addRouteViaInterface(fd, cidr, iface); err != nil {
				return routesAdded, fmt.Errorf("add IPv4 route %s: %w", cidr, err)
			}
			routesAdded++
		}

		for _, cidr := range gw.GetRoutesIPv6() {
			// TunnelNetworkPrefix is an IPv4 prefix ("10.255.24.") so this check
			// is a no-op for IPv6 CIDRs. It's kept for consistency with the IPv4
			// loop and as a safeguard in case the prefix changes in the future.
			if strings.HasPrefix(cidr, TunnelNetworkPrefix) {
				continue
			}
			if err := addRouteViaInterface(fd, cidr, iface); err != nil {
				return routesAdded, fmt.Errorf("add IPv6 route %s: %w", cidr, err)
			}
			routesAdded++
		}
	}

	return routesAdded, nil
}

func (c *DarwinConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.wgDevice != nil {
		return nil
	}

	tunDev, err := tun.CreateTUN(c.helperConfig.Interface, wireguardMTU)
	if err != nil {
		return fmt.Errorf("create TUN device: %w", err)
	}

	ifaceName, err := tunDev.Name()
	if err != nil {
		_ = tunDev.Close()
		return fmt.Errorf("get TUN device name: %w", err)
	}

	logger := &device.Logger{
		Verbosef: device.DiscardLogf,
		Errorf:   func(format string, args ...any) { fmt.Fprintf(os.Stderr, "wireguard: "+format+"\n", args...) },
	}
	wgDev := device.NewDevice(tunDev, conn.NewDefaultBind(), logger)

	if err := wgDev.Up(); err != nil {
		wgDev.Close()
		return fmt.Errorf("bring up wireguard device: %w", err)
	}

	// UAPI socket allows wgctrl to configure the device
	fileUAPI, err := ipc.UAPIOpen(ifaceName)
	if err != nil {
		wgDev.Close()
		return fmt.Errorf("open UAPI socket: %w", err)
	}

	uapi, err := ipc.UAPIListen(ifaceName, fileUAPI)
	if err != nil {
		_ = fileUAPI.Close()
		wgDev.Close()
		return fmt.Errorf("listen on UAPI socket: %w", err)
	}

	c.wgDevice = wgDev
	c.tunDev = tunDev
	c.uapi = uapi
	c.ifaceName = ifaceName

	// Configure IP addresses and bring the interface up.
	// We shell out to ifconfig because macOS has no stable public API for assigning
	// addresses to utun interfaces — the required SIOCAIFADDR_IN6 / SIOCSIFADDR
	// ioctls are undocumented and vary across OS versions. ifconfig is the standard
	// tool used by the macOS networking stack itself.
	commands := [][]string{
		{"ifconfig", ifaceName, "inet", cfg.GetDeviceIPv4() + "/21", cfg.GetDeviceIPv4(), "alias"},
	}
	if cfg.GetDeviceIPv6() != "" {
		commands = append(commands, []string{"ifconfig", ifaceName, "inet6", cfg.GetDeviceIPv6() + "/64", "alias"})
	}
	commands = append(commands, []string{"ifconfig", ifaceName, "up"})

	for _, s := range commands {
		cmd := exec.CommandContext(ctx, s[0], s[1:]...)

		if out, err := cmd.CombinedOutput(); err != nil {
			c.closeLocked()
			return fmt.Errorf("running %v: %w: %v", cmd, err, string(out))
		}

		// Small delay between ifconfig calls to avoid kernel ENOMEM / EBUSY
		// errors when rapidly configuring addresses on a freshly-created utun.
		time.Sleep(100 * time.Millisecond)
	}

	// Add /21 route for the tunnel network
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		c.closeLocked()
		return fmt.Errorf("lookup interface %q: %w", ifaceName, err)
	}

	routeFd, err := unix.Socket(unix.AF_ROUTE, unix.SOCK_RAW, unix.AF_UNSPEC)
	if err != nil {
		c.closeLocked()
		return fmt.Errorf("open routing socket: %w", err)
	}
	defer func() { _ = unix.Close(routeFd) }()

	if err := addRouteViaInterface(routeFd, cfg.GetDeviceIPv4()+"/21", iface); err != nil {
		c.closeLocked()
		return fmt.Errorf("add tunnel network route: %w", err)
	}

	// Start accepting UAPI connections only after all initialization has succeeded.
	// This ensures wgctrl can't race against incomplete interface setup.
	go func() {
		for {
			uapiConn, err := uapi.Accept()
			if err != nil {
				return
			}
			go wgDev.IpcHandle(uapiConn)
		}
	}()

	return nil
}

func (c *DarwinConfigurator) TeardownInterface(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.wgDevice == nil {
		return nil
	}

	c.closeLocked()
	return nil
}

// closeLocked shuts down the UAPI listener, WireGuard device, and TUN.
// Must be called with c.mu held.
func (c *DarwinConfigurator) closeLocked() {
	if c.uapi != nil {
		_ = c.uapi.Close()
		c.uapi = nil
	}
	if c.wgDevice != nil {
		c.wgDevice.Close()
		c.wgDevice = nil
	}
	// wgDevice.Close() also closes tunDev
	c.tunDev = nil
	c.ifaceName = ""
}

// addRouteViaInterface adds a route for the given CIDR through the specified
// network interface using the provided BSD routing socket fd.
func addRouteViaInterface(fd int, cidr string, iface *net.Interface) error {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return fmt.Errorf("parse CIDR %q: %w", cidr, err)
	}

	addrs := make([]route.Addr, syscall.RTAX_MAX)

	if prefix.Addr().Is6() {
		dst := prefix.Addr().As16()
		addrs[syscall.RTAX_DST] = &route.Inet6Addr{IP: dst}

		var mask [16]byte
		ones := prefix.Bits()
		for i := range ones {
			mask[i/8] |= 1 << (7 - i%8)
		}
		addrs[syscall.RTAX_NETMASK] = &route.Inet6Addr{IP: mask}
	} else {
		dst := prefix.Addr().As4()
		addrs[syscall.RTAX_DST] = &route.Inet4Addr{IP: dst}

		var mask [4]byte
		ones := prefix.Bits()
		for i := range ones {
			mask[i/8] |= 1 << (7 - i%8)
		}
		addrs[syscall.RTAX_NETMASK] = &route.Inet4Addr{IP: mask}
	}

	// LinkAddr as gateway routes directly through the interface
	addrs[syscall.RTAX_GATEWAY] = &route.LinkAddr{
		Index: iface.Index,
		Name:  iface.Name,
	}

	rtm := &route.RouteMessage{
		Version: syscall.RTM_VERSION,
		Type:    syscall.RTM_ADD,
		Flags:   syscall.RTF_UP | syscall.RTF_STATIC,
		Index:   iface.Index,
		ID:      uintptr(os.Getpid()),
		Seq:     int(time.Now().UnixNano() & 0x7fffffff),
		Addrs:   addrs,
	}

	b, err := rtm.Marshal()
	if err != nil {
		return fmt.Errorf("marshal route message: %w", err)
	}

	_, err = unix.Write(fd, b)
	if err != nil {
		if errors.Is(err, unix.EEXIST) {
			return nil
		}
		return fmt.Errorf("write route message: %w", err)
	}

	return nil
}
