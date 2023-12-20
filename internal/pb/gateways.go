package pb

func (x *Gateway) MergeHealth(y *Gateway) {
	x.Healthy = y.GetHealthy()
}

// MergeGatewayHealth copies the `Healthy` member from the existingGateways slice (if available), and returns updated new gateways.
func MergeGatewayHealth(existingGateways []*Gateway, newGateways []*Gateway) []*Gateway {
	var updatedGateways []*Gateway
	for _, newGateway := range newGateways {
		for _, existingGw := range existingGateways {
			if existingGw.GetName() == newGateway.GetName() {
				newGateway.Healthy = existingGw.GetHealthy()
				break
			}
		}

		updatedGateways = append(updatedGateways, newGateway)
	}

	return updatedGateways
}

// Satisfy WireGuard interface.
// IP addresses routed by a gateway includes configured routes plus the gateway itself.
func (x *Gateway) GetAllowedIPs() []string {
	ips := append(x.GetRoutesIPv4(), x.GetIpv4()+"/32")
	if x.GetIpv6() != "" {
		ips = append(ips, x.GetIpv6()+"/128")
		ips = append(ips, x.GetRoutesIPv6()...)
	}
	return ips
}
