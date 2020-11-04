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
	Owner     string `json:"owner"`
}

// GatewayInfo is the info provided by the gateway-agent in order to bootstrap a gateway
type GatewayInfo struct {
	Name     string `json:"name"`
	PublicIP string `json:"endpoint"`
}

// GatewayConfig is the response from the bootstrap-api, produced by the api server and used by the gateway-agent to connect
type GatewayConfig struct {
	Name               string `json:"name"`
	TunnelIP           string `json:"deviceIP"`
	APIServerPublicKey string `json:"apiServerPublicKey"`
	APIServerIP        string `json:"apiServerIP"`
}
