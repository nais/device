package config

type Config struct {
	DbConnURI   string
	SlackToken  string
	BindAddress string
}

func DefaultConfig() Config {
	return Config{}
}
