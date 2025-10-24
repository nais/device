package pb

import "slices"

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

// GetAllowedIPs returns the IP addresses routed by a gateway, including configured routes plus the gateway itself.
func (x *Gateway) GetAllowedIPs() []string {
	ips := append(x.GetRoutesIPv4(), x.GetIpv4()+"/32")
	if x.GetIpv6() != "" {
		ips = append(ips, x.GetIpv6()+"/128")
		ips = append(ips, x.GetRoutesIPv6()...)
	}
	return ips
}

func (x *Gateway) Equal(other *Gateway) bool {
	if x == other {
		return true
	}

	if x == nil || other == nil {
		return false
	}

	return x.GetName() == other.GetName() &&
		x.GetIpv4() == other.GetIpv4() &&
		x.GetIpv6() == other.GetIpv6() &&
		x.GetEndpoint() == other.GetEndpoint() &&
		x.GetPublicKey() == other.GetPublicKey() &&
		x.GetRequiresPrivilegedAccess() == other.GetRequiresPrivilegedAccess() &&
		slices.Equal(x.GetRoutesIPv4(), other.GetRoutesIPv4()) &&
		slices.Equal(x.GetRoutesIPv6(), other.GetRoutesIPv6()) &&
		slices.Equal(x.GetAllowedIPs(), other.GetAllowedIPs()) &&
		slices.Equal(x.GetAccessGroupIDs(), other.GetAccessGroupIDs())
}
