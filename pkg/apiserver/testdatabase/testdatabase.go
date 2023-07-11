package testdatabase

import (
	"context"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/nais/device/pkg/apiserver/database"
)

const (
	timeout                 = 5 * time.Second
	wireguardNetworkAddress = "10.255.240.1/21"
	apiserverWireGuardIP    = "10.255.240.1"
)

func Setup(t *testing.T) database.APIServer {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tempFile, err := os.CreateTemp(os.TempDir(), "*.sqlite")
	if err != nil {
		t.Fatalf("unable to setup test database: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			t.Logf("unable to clean up test database: %v", err)
		}
	})

	ipAllocator := database.NewIPAllocator(netip.MustParsePrefix(wireguardNetworkAddress), []string{apiserverWireGuardIP})
	db, err := database.New(ctx, tempFile.Name(), ipAllocator, false)
	if err != nil {
		t.Fatalf("Instantiating database: %v", err)
	}

	return db
}
