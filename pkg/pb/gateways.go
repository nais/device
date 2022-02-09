package pb

import (
	"fmt"
	"io"
	"strings"

	"github.com/nais/device/pkg/ioconvenience"
)

func (x *Gateway) MergeHealth(y *Gateway) {
	x.Healthy = y.GetHealthy()
}

// MergeGatewayHealth copies the `Healthy` member from one slice of gateways to the other.
func MergeGatewayHealth(dst []*Gateway, src []*Gateway) {
	gatewayByName := func(name string) *Gateway {
		for _, gw := range src {
			if gw.GetName() == name {
				return gw
			}
		}
		return nil
	}
	for _, gw := range dst {
		healthGateway := gatewayByName(gw.Name)
		if healthGateway != nil {
			gw.Healthy = healthGateway.GetHealthy()
		}
	}
}

func (x *Gateway) WritePeerConfig(w io.Writer) error {
	ew := ioconvenience.NewErrorWriter(w)

	routes := append(x.GetRoutes(), x.GetIp())
	allowedIPs := strings.Join(routes, ",")

	_, _ = io.WriteString(ew, "[Peer]\n")
	_, _ = io.WriteString(ew, fmt.Sprintf("PublicKey = %s\n", x.GetPublicKey()))
	_, _ = io.WriteString(ew, fmt.Sprintf("AllowedIPs = %s\n", allowedIPs))
	_, _ = io.WriteString(ew, fmt.Sprintf("Endpoint = %s\n", x.GetEndpoint()))
	_, _ = io.WriteString(ew, "\n")

	_, err := ew.Status()
	return err
}

func (x *Gateway) GetAllowedIPs() []string {
	return append(x.GetRoutes(), x.GetIp())
}
