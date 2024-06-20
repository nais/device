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

func (d *Gateway) Equal(other *Gateway) bool {
	if d == other {
		return true
	}

	if d == nil || other == nil {
		return false
	}

	return d.GetName() == other.GetName() &&
		d.GetIpv4() == other.GetIpv4() &&
		d.GetIpv6() == other.GetIpv6() &&
		d.GetEndpoint() == other.GetEndpoint() &&
		d.GetPublicKey() == other.GetPublicKey() &&
		d.GetRequiresPrivilegedAccess() == other.GetRequiresPrivilegedAccess() &&
		slices.Equal(d.GetRoutesIPv4(), other.GetRoutesIPv4()) &&
		slices.Equal(d.GetRoutesIPv6(), other.GetRoutesIPv6()) &&
		slices.Equal(d.GetAllowedIPs(), other.GetAllowedIPs()) &&
		slices.Equal(d.GetAccessGroupIDs(), other.GetAccessGroupIDs())
}
