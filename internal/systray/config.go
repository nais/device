package systray

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nais/device/internal/notify"
)

type Config struct {
	GrpcAddress string `json:"-"`

	ConfigDir string `json:"-"`

	LogLevel    string
	LogFilePath string `json:"-"`

	BlackAndWhiteIcons bool
	Notifier           notify.Notifier `json:"-"`
}

func (cfg *Config) Persist() error {
	configFile, err := os.Create(filepath.Join(cfg.ConfigDir, ConfigFile))
	if err != nil {
		return fmt.Errorf("opening file: %v", err)
	}

	err = json.NewEncoder(configFile).Encode(cfg)
	if err != nil {
		return fmt.Errorf("encoding json to file: %v", err)
	}
	return nil
}

func (cfg *Config) Populate() error {
	configFile, err := os.Open(filepath.Join(cfg.ConfigDir, ConfigFile))
	if err != nil {
		return fmt.Errorf("opening file: %v", err)
	}

	err = json.NewDecoder(configFile).Decode(&cfg)
	if err != nil {
		return fmt.Errorf("decoding json from file: %v", err)
	}

	return nil
}
