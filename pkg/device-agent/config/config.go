package config

import (
	"errors"
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
	AgentConfiguration       *pb.AgentConfiguration
	BootstrapAPI             string
	BootstrapToken           string
	ConfigDir                string
	DeviceAgentHelperAddress string
	EnableGoogleAuth         bool
	GoogleAuthServerAddress  string
	GrpcAddress              string
	Interface                string
	LogFilePath              string
	LogLevel                 string
	OAuth2Config             oauth2.Config
	OuttuneEnabled           bool
	PartnerDomain            string
	Platform                 string
	PrivateKeyPath           string
	SerialPath               string
	WireGuardBinary          string
	WireGuardConfigPath      string
	WireGuardGoBinary        string
	EnrollProjectID          string
	EnrollTopicName          string
}

func (c *Config) SetDefaults() {
	c.Platform = Platform
	c.SetPlatformDefaults()
	c.PrivateKeyPath = filepath.Join(c.ConfigDir, "private.key")
	c.WireGuardConfigPath = filepath.Join(c.ConfigDir, c.Interface+".conf")
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
		GoogleAuthServerAddress:  "https://naisdevice-auth-server-h2pjqrstja-lz.a.run.app",
		OuttuneEnabled:           true,
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

	if err := os.WriteFile(agentConfigPath, out, 0o644); err != nil {
		log.Errorf("Write AgentConfiguration: %v", err)
	}
}

func (c *Config) PopulateAgentConfiguration() {
	agentConfigPath := filepath.Join(c.ConfigDir, File)
	in, err := os.ReadFile(agentConfigPath)
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
