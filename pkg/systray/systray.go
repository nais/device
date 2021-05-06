package systray

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/getlantern/systray"

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


func WriteToJSONFile(strct interface{}, path string) error {
	b, err := json.Marshal(&strct)
	if err != nil {
		return fmt.Errorf("marshaling struct into json: %w", err)
	}
	if err := ioutil.WriteFile(path, b, 0600); err != nil {
		return err
	}

	log.Infof("Wrote config %+v to %s", strct, path)
	return nil
}


func ReadFromJSONFile(path string) (*Config, error) {
	var config Config
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading systray config from disk: %w", err)
	}
	if err := json.Unmarshal(b, &config); err != nil {
		return nil, fmt.Errorf("unmarshaling systray config: %w", err)
	}
	return &config, nil
}
