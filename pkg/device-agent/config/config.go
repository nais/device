package config

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protojson"

	config2 "github.com/nais/device/pkg/helper/config"
	"github.com/nais/device/pkg/pb"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/nais/device/pkg/config"
)

const File = "agent-config.json"

type Config struct {
	APIServer                string
	APIServerGRPCAddress     string
	Interface                string
	ConfigDir                string
	BootstrapToken           string
	WireGuardBinary          string
	WireGuardGoBinary        string
	PrivateKeyPath           string
	WireGuardConfigPath      string
	BootstrapConfigPath      string
	SerialPath               string
	LogLevel                 string
	LogFilePath              string
	OAuth2Config             oauth2.Config
	Platform                 string
	BootstrapAPI             string
	GrpcAddress              string
	DeviceAgentHelperAddress string
	AgentConfiguration       *pb.AgentConfiguration
	EnableGoogleAuth         bool
}

func (c *Config) SetDefaults() {
	c.Platform = Platform
	c.SetPlatformDefaults()
	c.PrivateKeyPath = filepath.Join(c.ConfigDir, "private.key")
	c.WireGuardConfigPath = filepath.Join(c.ConfigDir, c.Interface+".conf")
	c.BootstrapConfigPath = filepath.Join(c.ConfigDir, "bootstrapconfig.json")
	c.SerialPath = filepath.Join(c.ConfigDir, "product_serial")
}

func DefaultConfig() Config {
	userConfigDir, err := config.UserConfigDir()
	if err != nil {
		log.Fatal("Getting user config dir: %w", err)
	}

	return Config{
		APIServer:                "http://10.255.240.1",
		APIServerGRPCAddress:     "10.255.240.1:8099",
		BootstrapAPI:             "https://bootstrap.device.nais.io",
		ConfigDir:                userConfigDir,
		LogLevel:                 "info",
		GrpcAddress:              filepath.Join(userConfigDir, "agent.sock"),
		DeviceAgentHelperAddress: filepath.Join(config2.RuntimeDir, "helper.sock"),
		OAuth2Config: oauth2.Config{
			ClientID:    "8086d321-c6d3-4398-87da-0d54e3d93967",
			Scopes:      []string{"openid", "6e45010d-2637-4a40-b91d-d4cbb451fb57/.default", "offline_access"},
			Endpoint:    endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
			RedirectURL: "http://localhost:PORT/",
		},
	}
}

func (c *Config) PersistAgentConfiguration() {
	agentConfigPath := filepath.Join(c.ConfigDir, File)

	out, err := protojson.Marshal(c.AgentConfiguration)
	if err != nil {
		log.Errorf("Encode AgentConfiguration: %v", err)
	}

	log.Debugf("persisting agent-config: %+v", c.AgentConfiguration)

	if err := ioutil.WriteFile(agentConfigPath, out, 0644); err != nil {
		log.Errorf("Write AgentConfiguration: %v", err)
	}
}

func (c *Config) PopulateAgentConfiguration() {
	agentConfigPath := filepath.Join(c.ConfigDir, File)
	in, err := ioutil.ReadFile(agentConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.AgentConfiguration = &pb.AgentConfiguration{}
			return
		}

		log.Errorf("Read AgentConfiguration: %v", err)
	}

	tempCfg := &pb.AgentConfiguration{}
	if err := protojson.Unmarshal(in, tempCfg); err != nil {
		log.Fatalln("Failed to parse address book:", err)
	}

	c.AgentConfiguration = tempCfg
	log.Debugf("read agent-config: %v", c.AgentConfiguration)
}
