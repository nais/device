package bootstrap

// Config is the information the device needs to bootstrap it's connection to the APIServer
type Config struct {
	DeviceIP       string `json:"deviceIP"`
	PublicKey      string `json:"publicKey"`
	TunnelEndpoint string `json:"tunnelEndpoint"`
	APIServerIP    string `json:"apiServerIP"`
}

// DeviceInfo is the information sent by the device during enrollment
type DeviceInfo struct {
	Serial    string `json:"serial"`
	PublicKey string `json:"publicKey"`
	Platform  string `json:"platform"`
}
