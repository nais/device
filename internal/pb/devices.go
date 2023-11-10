package pb

// Satisfy WireGuard interface.
// This value is written to the config file as a comment, so we put in the serial of the device in order to identify it.
func (x *Device) GetName() string {
	return x.GetSerial()
}

// Satisfy WireGuard interface.
// This field contains the private IP addresses of a device.
func (x *Device) GetAllowedIPs() []string {
	ips := []string{x.GetIpv4() + "/32"}
	if x.GetIpv6() != "" {
		ips = append(ips, x.GetIpv6()+"/128")
	}
	return ips
}

// Satisfy WireGuard interface.
// Endpoints are not used when configuring gateway and api server; connections are initiated from the client.
func (x *Device) GetEndpoint() string {
	return ""
}
