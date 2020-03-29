package config

type Config struct {
	DbConnURI string
}

func DefaultConfig() Config {
	return Config{}
}
