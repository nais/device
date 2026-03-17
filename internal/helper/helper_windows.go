package helper

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"

	"github.com/nais/device/internal/iputil"
	"github.com/nais/device/internal/wgconfig"
	"github.com/nais/device/pkg/pb"
)

type WindowsConfigurator struct {
	helperConfig Config

	mu        sync.Mutex
	wgDevice  *device.Device
	tunDev    tun.Device
	uapi      net.Listener
	tunnelNet netip.Prefix
}

var _ OSConfigurator = &WindowsConfigurator{}

func New(helperConfig Config, _ *logrus.Entry) *WindowsConfigurator {
	return &WindowsConfigurator{
		helperConfig: helperConfig,
	}
}

func (c *WindowsConfigurator) Prerequisites() error {
	return nil
}

func (c *WindowsConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	return wgconfig.ApplyConfig(ctx, c.helperConfig.Interface, cfg)
}

func (c *WindowsConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tunDev == nil {
		return 0, fmt.Errorf("TUN device not initialized")
	}

	nativeTun, ok := c.tunDev.(*tun.NativeTun)
	if !ok {
		return 0, fmt.Errorf("unexpected TUN device type %T", c.tunDev)
	}
	ifLUID := winipcfg.LUID(nativeTun.LUID())

	routesAdded := 0
	for _, gw := range gateways {
		for _, cidr := range append(gw.GetRoutesIPv4(), gw.GetRoutesIPv6()...) {
			cidr = strings.TrimSpace(cidr)

			if IsTunnelRoute(c.tunnelNet, cidr) {
				continue
			}

			dst, err := iputil.ParsePrefix(cidr)
			if err != nil {
				return routesAdded, fmt.Errorf("parse route: %w", err)
			}

			nextHop := netip.IPv4Unspecified()
			if dst.Addr().Is6() {
				nextHop = netip.IPv6Unspecified()
			}

			if err := ifLUID.AddRoute(dst, nextHop, 0); err != nil {
				if errors.Is(err, windows.ERROR_OBJECT_ALREADY_EXISTS) {
					continue
				}
				return routesAdded, fmt.Errorf("add route %s: %w", cidr, err)
			}
			routesAdded++
		}
	}

	return routesAdded, nil
}

func (c *WindowsConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	tunnelNet, err := TunnelNetworkFromIP(cfg.DeviceIPv4)
	if err != nil {
		return fmt.Errorf("derive tunnel network: %w", err)
	}
	c.tunnelNet = tunnelNet

	if c.wgDevice != nil {
		return nil
	}

	tunDev, err := tun.CreateTUN(c.helperConfig.Interface, wireguardMTU)
	if err != nil {
		return fmt.Errorf("create TUN device: %w", err)
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

	uapi, err := ipc.UAPIListen(c.helperConfig.Interface)
	if err != nil {
		wgDev.Close()
		return fmt.Errorf("listen on UAPI named pipe: %w", err)
	}

	c.wgDevice = wgDev
	c.tunDev = tunDev
	c.uapi = uapi

	nativeTun, ok := tunDev.(*tun.NativeTun)
	if !ok {
		c.closeLocked()
		return fmt.Errorf("unexpected TUN device type %T", tunDev)
	}
	ifLUID := winipcfg.LUID(nativeTun.LUID())

	ipv4, err := netip.ParsePrefix(cfg.DeviceIPv4 + "/21")
	if err != nil {
		c.closeLocked()
		return fmt.Errorf("parse IPv4 address: %w", err)
	}

	if err := ifLUID.AddIPAddress(ipv4); err != nil {
		c.closeLocked()
		return fmt.Errorf("add IPv4 address: %w", err)
	}

	if cfg.DeviceIPv6 != "" {
		ipv6, err := netip.ParsePrefix(cfg.DeviceIPv6 + "/64")
		if err != nil {
			c.closeLocked()
			return fmt.Errorf("parse IPv6 address: %w", err)
		}

		if err := ifLUID.AddIPAddress(ipv6); err != nil {
			c.closeLocked()
			return fmt.Errorf("add IPv6 address: %w", err)
		}
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

func (c *WindowsConfigurator) TeardownInterface(ctx context.Context) error {
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
func (c *WindowsConfigurator) closeLocked() {
	if c.uapi != nil {
		_ = c.uapi.Close()
		c.uapi = nil
	}
	if c.wgDevice != nil {
		c.wgDevice.Close()
		c.wgDevice = nil
	}
	c.tunDev = nil
}
