package systray

import (
	"context"

	"github.com/getlantern/systray"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/pb"
	log "github.com/sirupsen/logrus"
)

var connection *grpc.ClientConn

const ConfigFile = "systray-config.json"

func onReady(ctx context.Context, cfg Config, notifier notify.Notifier) {
	log.Debugf("naisdevice-agent on unix socket %s", cfg.GrpcAddress)
	connection, err := grpc.Dial(
		"unix:"+cfg.GrpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("unable to connect to naisdevice-agent grpc server: %v", err)
	}

	client := pb.NewDeviceAgentClient(connection)

	gui := NewGUI(ctx, client, cfg, notifier)

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

func Spawn(ctx context.Context, systrayConfig Config, notifier notify.Notifier) {
	systray.Run(func() { onReady(ctx, systrayConfig, notifier) }, onExit)
}
