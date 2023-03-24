package systray

import (
	"context"

	"fyne.io/systray"
	"google.golang.org/grpc"

	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

var connection *grpc.ClientConn

const ConfigFile = "systray-config.json"

func onReady(ctx context.Context, cfg Config) {
	log.Debugf("naisdevice-agent on unix socket %s", cfg.GrpcAddress)
	connection, err := grpc.Dial(
		"unix:"+cfg.GrpcAddress,
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("unable to connect to naisdevice-agent grpc server: %v", err)
	}

	client := pb.NewDeviceAgentClient(connection)

	gui := NewGUI(ctx, client, cfg)

	go gui.handleStatusStream(ctx)
	go gui.handleButtonClicks(ctx)
	go gui.EventLoop(ctx)
	// TODO: go checkVersion(versionCheckInterval, gui)
}

// This is where we clean up
func onExit() {
	log.Infof("Shutting down.")

	if connection != nil {
		connection.Close()
	}
}

func Spawn(ctx context.Context, systrayConfig Config) {
	systray.Run(func() { onReady(ctx, systrayConfig) }, onExit)
}
