package systray

import (
	"fmt"
	"github.com/getlantern/systray"
	"net"

	"google.golang.org/grpc"

	pb "github.com/nais/device/pkg/protobuf"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	GrpcPort   uint16
	GrpcServer net.IP

	ConfigDir string

	LogLevel    string
	LogFilePath string

	AutoConnect bool
}

var cfg Config

func onReady() {
	connection, err := grpc.Dial(
		fmt.Sprintf("127.0.0.1:%d", cfg.GrpcPort),
		grpc.WithBlock(),
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

	go gui.handleButtonClicks()
	go gui.EventLoop()
	// TODO: go checkVersion(versionCheckInterval, gui)
}

func onExit() {
	// This is where we clean up
}

func Spawn(systrayConfig Config) {
	cfg = systrayConfig

	systray.Run(onReady, onExit)
}
