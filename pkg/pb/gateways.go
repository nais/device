package pb

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

// Satisfy WireGuard interface.
// IP addresses routed by a gateway includes configured routes plus the gateway itself.
func (x *Gateway) GetAllowedIPs() []string {
	return append(x.GetRoutes(), x.GetIp())
}
