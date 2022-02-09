package pb

import (
	"fmt"
	"io"
	"time"

	"github.com/nais/device/pkg/ioconvenience"
)

const MaxTimeSinceKolideLastSeen = 24 * time.Hour

func (d *Device) KolideSeenRecently() bool {
	lastSeen := d.GetKolideLastSeen().AsTime()
	deadline := lastSeen.Add(MaxTimeSinceKolideLastSeen)

	return deadline.After(time.Now())
}

func (d *Device) WritePeerConfig(w io.Writer) error {
	ew := ioconvenience.NewErrorWriter(w)

	_, _ = io.WriteString(ew, "[Peer]\n")
	_, _ = io.WriteString(ew, fmt.Sprintf("PublicKey = %s\n", d.GetPublicKey()))
	_, _ = io.WriteString(ew, fmt.Sprintf("AllowedIPs = %s\n", d.GetIp()))
	_, _ = io.WriteString(ew, "\n")

	_, err := ew.Status()
	return err
}

func DevicesAsPeers(devices []*Device) []Peer {
	peers := make([]Peer, len(devices))
	for i := range peers {
		peers[i] = devices[i]
	}
	return peers
}
