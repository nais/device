package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protojson"

	config2 "github.com/nais/device/internal/helper/config"
	"github.com/nais/device/pkg/pb"

	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/nais/device/pkg/config"
)

const File = "agent-config.json"

type Config struct {
	AgentConfiguration       *pb.AgentConfiguration
	ConfigDir                string
	DeviceAgentHelperAddress string
	GoogleAuthServerAddress  string
	GrpcAddress              string
	Interface                string
	LogFilePath              string
	LogLevel                 string
	AzureOAuth2Config        oauth2.Config
	GoogleOAuth2Config       oauth2.Config
	Platform                 string
	PrivateKeyPath           string
	WireGuardConfigPath      string
	EnrollProjectID          string
	EnrollTopicName          string
	LocalAPIServer           bool
}

func (c *Config) SetDefaults() {
	c.Platform = Platform
	c.Interface = "utun69"
	c.PrivateKeyPath = filepath.Join(c.ConfigDir, "private.key")
	c.WireGuardConfigPath = filepath.Join(c.ConfigDir, c.Interface+".conf")
}

func DefaultConfig() (*Config, error) {
	userConfigDir, err := config.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("getting user config dir: %w", err)
	}

	return &Config{
		ConfigDir:                userConfigDir,
		LogLevel:                 "info",
		GrpcAddress:              filepath.Join(userConfigDir, "agent.sock"),
		DeviceAgentHelperAddress: filepath.Join(config2.RuntimeDir, "helper.sock"),
		GoogleAuthServerAddress:  "https://naisdevice-auth-server-h2pjqrstja-lz.a.run.app",
		AzureOAuth2Config: oauth2.Config{
			ClientID: "8086d321-c6d3-4398-87da-0d54e3d93967",
			Scopes: []string{
				"openid",
				"6e45010d-2637-4a40-b91d-d4cbb451fb57/.default",
				"offline_access",
			},
			Endpoint:    endpoints.AzureAD("62366534-1ec3-4962-8869-9b5535279d0b"),
			RedirectURL: "http://localhost:PORT/",
		},
		GoogleOAuth2Config: oauth2.Config{
			ClientID:    "955023559628-g51n36t4icbd6lq7ils4r0ol9oo8kpk0.apps.googleusercontent.com",
			Scopes:      []string{"https://www.googleapis.com/auth/userinfo.email"},
			Endpoint:    endpoints.Google,
			RedirectURL: "http://localhost:PORT/google",
		},
	}, nil
}

func (c *Config) OAuth2Config(provider pb.AuthProvider) oauth2.Config {
	if provider == pb.AuthProvider_Google {
		return c.GoogleOAuth2Config
	}
	return c.AzureOAuth2Config
}

func (c *Config) PersistAgentConfiguration(log *logrus.Entry) {
	agentConfigPath := filepath.Join(c.ConfigDir, File)

	out, err := protojson.MarshalOptions{Indent: "  "}.Marshal(c.AgentConfiguration)
	if err != nil {
		log.WithError(err).Error("encode AgentConfiguration")
	}

	log.WithField("cfg", c.AgentConfiguration).Debug("persisting agent-config")

	if err := os.WriteFile(agentConfigPath, out, 0o644); err != nil {
		log.WithError(err).Error("write AgentConfiguration")
	}
}

func (c *Config) PopulateAgentConfiguration(log *logrus.Entry) {
	agentConfigPath := filepath.Join(c.ConfigDir, File)
	in, err := os.ReadFile(agentConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.AgentConfiguration = &pb.AgentConfiguration{}
			return
		}

		log.WithError(err).Error("read AgentConfiguration")
	}

	tempCfg := &pb.AgentConfiguration{}
	if err := protojson.Unmarshal(in, tempCfg); err != nil {
		log.WithError(err).Fatal("failed to parse agent config")
	}

	c.AgentConfiguration = tempCfg
	c.PersistAgentConfiguration(log)

	log.WithField("cfg", c.AgentConfiguration).Debug("read agent-config")
}
