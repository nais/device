package pb

// Satisfy WireGuard interface.
// This value is written to the config file as a comment, so we put in the serial of the device in order to identify it.
func (x *Device) GetName() string {
	return x.GetSerial()
}

// Satisfy WireGuard interface.
// This field contains the private IP address of a device.
func (x *Device) GetAllowedIPs() []string {
	return []string{x.GetIp()}
}

// Satisfy WireGuard interface.
// Endpoints are not used when configuring gateway and api server; connections are initiated from the client.
func (x *Device) GetEndpoint() string {
	return ""
}
