// +build linux darwin

package wireguard

import (
	"fmt"
	"io"

	"github.com/nais/device/pkg/pb"
)

var wireGuardTemplateHeader = `[Interface]
PrivateKey = %s

`

func MarshalHeader(w io.Writer, x *pb.Configuration) (int, error) {
	return fmt.Fprintf(w, wireGuardTemplateHeader, x.GetPrivateKey())
}
