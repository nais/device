package pb

import (
	"fmt"
	"io"
	"time"

	"github.com/nais/device/pkg/ioconvenience"
)

const MaxTimeSinceKolideLastSeen = 24 * time.Hour

func (x *Device) KolideSeenRecently() bool {
	lastSeen := x.GetKolideLastSeen().AsTime()
	deadline := lastSeen.Add(MaxTimeSinceKolideLastSeen)

	return deadline.After(time.Now())
}

func (x *Device) WritePeerConfig(w io.Writer) error {
	ew := ioconvenience.NewErrorWriter(w)

	_, _ = io.WriteString(ew, "[Peer]\n")
	_, _ = io.WriteString(ew, fmt.Sprintf("PublicKey = %s\n", x.GetPublicKey()))
	_, _ = io.WriteString(ew, fmt.Sprintf("AllowedIPs = %s\n", x.GetIp()))
	_, _ = io.WriteString(ew, "\n")

	_, err := ew.Status()
	return err
}

// Satisfy WireGuard interface.
func (x *Device) GetName() string {
	return x.GetSerial()
}

// Satisfy WireGuard interface.
func (x *Device) GetAllowedIPs() []string {
	return []string{x.GetIp()}
}

// Satisfy WireGuard interface.
// Endpoints are not used when configuring gateway and api server; connections are initiated from the client.
func (x *Device) GetEndpoint() string {
	return ""
}

func DevicesAsPeers(devices []*Device) []Peer {
	peers := make([]Peer, len(devices))
	for i := range peers {
		peers[i] = devices[i]
	}
	return peers
}
