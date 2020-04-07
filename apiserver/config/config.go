package config

type Config struct {
	DbConnURI   string
	BindAddress string
}

func DefaultConfig() Config {
	return Config{
		BindAddress: "10.255.240.1:80",
	}
}
