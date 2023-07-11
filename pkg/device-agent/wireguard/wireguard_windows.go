package wireguard

import (
	"fmt"
	"io"

	"github.com/nais/device/pkg/pb"
)

/*
On windows we use "WireGuard-windows" client, which is basically a GUI wrapper of wg-quick. This config file requires
MTU and Address as additional fields because this also sets up the WireGuard interface for us.
*/
var wireGuardTemplateHeader = `[Interface]
PrivateKey = %s
MTU = %d
Address = %s
`

const (
	windows = true
	mtu     = 1360
)

func MarshalHeader(w io.Writer, x *pb.Configuration) (int, error) {
	return fmt.Fprintf(w, wireGuardTemplateHeader, x.GetPrivateKey(), mtu, x.GetDeviceIP())
}
