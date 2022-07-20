package systray

import (
	"encoding/json"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	GrpcAddress string

	ConfigDir string

	LogLevel    string
	LogFilePath string

	BlackAndWhiteIcons bool
	AgentPath          string `json:"-"`
}

func (cfg *Config) Persist() {
	configFile, err := os.Create(filepath.Join(cfg.ConfigDir, ConfigFile))
	if err != nil {
		log.Infof("opening file: %v", err)
	}

	err = json.NewEncoder(configFile).Encode(cfg)
	if err != nil {
		log.Warnf("encoding json to file: %v", err)
	}
}

func (cfg *Config) Populate() {
	var tempCfg Config

	configFile, err := os.Open(filepath.Join(cfg.ConfigDir, ConfigFile))
	if err != nil {
		log.Infof("opening file: %v", err)
	}

	err = json.NewDecoder(configFile).Decode(&tempCfg)
	if err != nil {
		log.Warnf("decoding json from file: %v", err)
		return
	}

	*cfg = tempCfg
}
