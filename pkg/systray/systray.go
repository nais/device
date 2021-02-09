package systray

import (
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
}

var cfg Config

var connection *grpc.ClientConn

func onReady() {
	var err error

	log.Infof("connecting to device-agent on unix socket %s...", cfg.GrpcAddress)
	connection, err = grpc.Dial(
		"unix:"+cfg.GrpcAddress,
		grpc.WithBlock(),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("unable to connect to naisdevice-agent grpc server: %v", err)
	}

	log.Info("connection to device-agent established")
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
	if connection != nil {
		connection.Close()
	}
}

func Spawn(systrayConfig Config) {
	cfg = systrayConfig

	systray.Run(onReady, onExit)
}
