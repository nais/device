package testdatabase

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/ip"
	"github.com/sirupsen/logrus"
)

const (
	timeout                 = 5 * time.Second
	wireguardNetworkAddress = "10.255.240.1/21"
	apiserverWireGuardIP    = "10.255.240.1"
)

func Setup(t *testing.T, kolideEnabled bool) database.Database {
	testDir := filepath.Join(os.TempDir(), "naisdevice-tests")
	err := os.MkdirAll(testDir, 0o755)
	if err != nil {
		t.Fatalf("unable to setup test dir for database tests: %v", err)
	}
	tempFile, err := os.CreateTemp(testDir, fmt.Sprintf("%s*.db", strings.ReplaceAll(t.Name(), "/", "_")))
	if err != nil {
		t.Fatalf("unable to setup test database: %v", err)
	}
	t.Logf("created database in: %v", tempFile.Name())
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("test failed, leaving test database in: %v", tempFile.Name())
		} else {
			for _, ext := range []string{"", "-wal", "-shm"} {
				if err := os.Remove(tempFile.Name() + ext); err != nil {
					t.Logf("unable to clean up test database: %v: %v", tempFile.Name(), err)
				} else {
					t.Logf("cleaned up test database: %v", tempFile.Name())
				}
			}
		}
	})

	ipAllocator := ip.NewV4Allocator(netip.MustParsePrefix(wireguardNetworkAddress), []string{apiserverWireGuardIP})
	prefix := netip.MustParsePrefix("fd00::/64")
	ip6Allocator := ip.NewV6Allocator(&prefix)
	db, err := database.New(tempFile.Name(), ipAllocator, ip6Allocator, kolideEnabled, logrus.New())
	if err != nil {
		t.Fatalf("Instantiating database: %v", err)
	}

	return db
}
