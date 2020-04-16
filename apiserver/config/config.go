package config

type Config struct {
	DbConnURI           string
	SlackToken          string
	BindAddress         string
	ConfigDir           string
	PrivateKeyPath      string
	WireGuardConfigPath string
	SkipSetupInterface  bool
	Endpoint            string
}

func DefaultConfig() Config {
	return Config{
		BindAddress: "10.255.240.1:80",
		ConfigDir:   "/usr/local/etc/nais-device/",
	}
}
