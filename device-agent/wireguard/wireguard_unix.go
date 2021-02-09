// +build linux darwin

package wireguard

import (
	"encoding/base64"
	"fmt"
	"io"

	"github.com/nais/device/pkg/pb"
)

var wireGuardTemplateHeader = `[Interface]
PrivateKey = %s

`

func MarshalHeader(w io.Writer, x *pb.Configuration) (int, error) {
	return fmt.Fprintf(w, wireGuardTemplateHeader, base64.StdEncoding.EncodeToString([]byte(x.GetPrivateKey())))
}

