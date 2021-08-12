package systray

import (
	"context"

	"github.com/getlantern/systray"
	"google.golang.org/grpc"

	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

var connection *grpc.ClientConn

const ConfigFile = "systray-config.json"

func onReady(cfg Config) {
	programContext := context.Background()
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

	gui := NewGUI(programContext, client, cfg)

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
	systray.Run(func() { onReady(systrayConfig) }, onExit)
}
