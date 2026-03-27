package helper

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"

	"github.com/nais/device/internal/ioconvenience"
	"github.com/nais/device/internal/iputil"
	"github.com/sirupsen/logrus"
)

const (
	// wireguardMTU is the MTU used for WireGuard tunnel interfaces across all platforms.
	wireguardMTU = 1360

	// tunnelIPv4PrefixLen is the prefix length used for the IPv4 tunnel network.
	tunnelIPv4PrefixLen = 21
)

// TunnelNetworkFromIP derives the tunnel network prefix from the device's IPv4 address.
func TunnelNetworkFromIP(deviceIPv4 string) (netip.Prefix, error) {
	addr, err := netip.ParseAddr(deviceIPv4)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("parse device IPv4 %q: %w", deviceIPv4, err)
	}
	return netip.PrefixFrom(addr, tunnelIPv4PrefixLen).Masked(), nil
}

// IsTunnelRoute reports whether cidr falls within the tunnel network.
func IsTunnelRoute(tunnelNet netip.Prefix, cidr string) bool {
	prefix, err := iputil.ParsePrefix(cidr)
	if err != nil {
		return false
	}
	return tunnelNet.Contains(prefix.Addr()) && prefix.Bits() >= tunnelNet.Bits()
}

func ZipLogFiles(files []string, log logrus.FieldLogger) (string, error) {
	if len(files) == 0 {
		return "nil", errors.New("can't be bothered to zip nothing")
	}
	archive, err := os.CreateTemp(os.TempDir(), "naisdevice_logs.*.zip")
	if err != nil {
		return "nil", err
	}
	defer ioconvenience.CloseWithLog(archive, log)
	zipWriter := zip.NewWriter(archive)
	for _, filename := range files {
		_, err = os.Stat(filename)
		if os.IsNotExist(err) {
			continue
		}
		logFile, err := os.Open(filename)
		if err != nil {
			return "nil", fmt.Errorf("%s: %w", filename, err)
		}
		zipEntryWiter, err := zipWriter.Create(filepath.Base(filename))
		if err != nil {
			return "nil", err
		}
		if _, err := io.Copy(zipEntryWiter, logFile); err != nil {
			return "nil", err
		}
		err = logFile.Close()
		if err != nil {
			return "nil", err
		}
	}
	err = zipWriter.Close()
	if err != nil {
		return "nil", err
	}
	return archive.Name(), nil
}
