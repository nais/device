package systray

import (
	"encoding/json"
	"github.com/getlantern/systray"
	"os"
	"path/filepath"

	"google.golang.org/grpc"

	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	GrpcAddress string

	ConfigDir string

	LogLevel    string
	LogFilePath string

	AutoConnect bool
	BlackAndWhiteIcons bool
}

var cfg Config

var connection *grpc.ClientConn

const ConfigFile = "/systray-config.json"

func onReady() {
	var err error

	log.Debugf("naisdevice-agent on unix socket %s", cfg.GrpcAddress)
	connection, err = grpc.Dial(
		"unix:"+cfg.GrpcAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("unable to connect to naisdevice-agent grpc server: %v", err)
	}

	client := pb.NewDeviceAgentClient(connection)

	gui := NewGUI(client)
	if cfg.AutoConnect {
		gui.Events <- ConnectClicked
	}

	go gui.handleStatusStream()
	go gui.handleButtonClicks()
	go gui.EventLoop()
	// TODO: go checkVersion(versionCheckInterval, gui)
}

// This is where we clean up
func onExit() {
	log.Infof("Shutting down.")

	if connection != nil {
		connection.Close()
	}
}

func Spawn(systrayConfig Config) {
	cfg = systrayConfig

	systray.Run(onReady, onExit)
}

func (cfg *Config) Persist() {
	configFile, err := os.Create(filepath.Join(cfg.ConfigDir, "systray-config.json"))
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

	configFile, err := os.Open(filepath.Join(cfg.ConfigDir, "systray-config.json"))
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