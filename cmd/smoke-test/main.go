// smoke-test connects to the running naisdevice-helper via gRPC,
// sends a Configure request with synthetic WireGuard keys and a test
// gateway, then verifies that:
//   - the WireGuard interface has the expected peer
//   - the OS has routes for the gateway's advertised prefixes
//
// Finally it tears down the configuration and exits.
package main

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"os"
	"path/filepath"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nais/device/internal/helper/config"
	"github.com/nais/device/pkg/pb"
)

const ifaceName = "utun69"

var wantRoutes = []netip.Prefix{
	netip.MustParsePrefix("10.123.0.0/24"),
	netip.MustParsePrefix("10.124.0.0/16"),
}

func main() {
	log.SetOutput(os.Stderr)

	if err := run(); err != nil {
		log.Fatalf("FAIL: %v", err)
	}
	log.Println("PASS: smoke test succeeded")
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	socketPath := filepath.Join(config.RuntimeDir, "helper.sock")
	conn, err := grpc.NewClient(
		"unix:"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("create gRPC client: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client := pb.NewDeviceHelperClient(conn)

	log.Println("pinging helper...")
	if _, err := client.Ping(ctx, &pb.PingRequest{}); err != nil {
		return fmt.Errorf("ping helper: %w", err)
	}
	log.Println("helper is alive")

	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return fmt.Errorf("generate private key: %w", err)
	}

	gwPrivateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return fmt.Errorf("generate gateway private key: %w", err)
	}
	gwPublicKey := gwPrivateKey.PublicKey()

	cfg := &pb.Configuration{
		PrivateKey: privateKey.String(),
		DeviceIPv4: "10.255.24.100",
		Gateways: []*pb.Gateway{
			{
				Name:       "smoke-gw",
				PublicKey:  gwPublicKey.String(),
				Endpoint:   "127.0.0.1:51820",
				Ipv4:       "10.255.24.1",
				RoutesIPv4: []string{"10.123.0.0/24", "10.124.0.0/16"},
			},
		},
	}

	log.Println("sending Configure request...")
	if _, err := client.Configure(ctx, cfg); err != nil {
		return fmt.Errorf("configure helper: %w", err)
	}
	log.Println("configure succeeded")

	if err := verifyPeers(gwPublicKey); err != nil {
		return fmt.Errorf("verify peers: %w", err)
	}

	if err := verifyRoutes(wantRoutes); err != nil {
		return fmt.Errorf("verify routes: %w", err)
	}

	log.Println("sending Teardown request...")
	if _, err := client.Teardown(ctx, &pb.TeardownRequest{}); err != nil {
		return fmt.Errorf("teardown: %w", err)
	}
	log.Println("teardown succeeded")

	return nil
}

func verifyPeers(expectedPubKey wgtypes.Key) error {
	log.Println("verifying WireGuard peers...")

	wgClient, err := wgctrl.New()
	if err != nil {
		return fmt.Errorf("create wgctrl client: %w", err)
	}
	defer func() { _ = wgClient.Close() }()

	dev, err := wgClient.Device(ifaceName)
	if err != nil {
		return fmt.Errorf("get device %q: %w", ifaceName, err)
	}

	if len(dev.Peers) == 0 {
		return fmt.Errorf("interface %q has no peers", ifaceName)
	}

	found := false
	for _, peer := range dev.Peers {
		log.Printf("  peer: %s (endpoint=%v, allowed_ips=%v)", peer.PublicKey, peer.Endpoint, peer.AllowedIPs)
		if peer.PublicKey == expectedPubKey {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("expected peer %s not found on interface %q", expectedPubKey, ifaceName)
	}

	log.Printf("PASS: interface %q has %d peer(s), expected peer found", ifaceName, len(dev.Peers))
	return nil
}
