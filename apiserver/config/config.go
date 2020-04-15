package config

type Config struct {
	DbConnURI                string
	SlackToken               string
	BindAddress              string
	ConfigDir                string
	PrivateKeyPath           string
	ControlPlaneWGConfigPath string
	SkipSetupInterface       bool
	ControlPlaneEndpoint     string
}

func DefaultConfig() Config {
	return Config{
		BindAddress: "10.255.240.1:80",
		ConfigDir:   "/usr/local/etc/nais-device/",
	}
}
