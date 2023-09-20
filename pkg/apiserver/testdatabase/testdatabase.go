package testdatabase

import (
	"context"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/ip"
)

const (
	timeout                 = 5 * time.Second
	wireguardNetworkAddress = "10.255.240.1/21"
	apiserverWireGuardIP    = "10.255.240.1"
)

func Setup(t *testing.T) database.APIServer {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tempFile, err := os.CreateTemp(os.TempDir(), "*.db")
	if err != nil {
		t.Fatalf("unable to setup test database: %v", err)
	}
	t.Logf("Created tmp database in: %v", tempFile.Name())
	t.Cleanup(func() {
		if cleanErr := os.Remove(tempFile.Name()); cleanErr != nil {
			t.Logf("unable to clean up test database: %v", err)
		}
	})

	ipAllocator := ip.NewV4Allocator(netip.MustParsePrefix(wireguardNetworkAddress), []string{apiserverWireGuardIP})
	prefix := netip.MustParsePrefix("fd00::/64")
	ip6Allocator := ip.NewV6Allocator(&prefix)
	db, err := database.New(ctx, tempFile.Name(), ipAllocator, ip6Allocator, false)
	if err != nil {
		t.Fatalf("Instantiating database: %v", err)
	}

	return db
}
