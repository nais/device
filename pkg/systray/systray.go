package systray

import (
	"context"
	"fmt"
	"github.com/gen2brain/beeep"

	"github.com/getlantern/systray"
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

type Announcement interface {
	Notify(format string, args ...any)
}

type logFunc func(string, ...any)

func logfn(logLevel log.Level) logFunc {
	switch logLevel {
	case log.InfoLevel:
		return log.Infof
	case log.ErrorLevel:
		return log.Errorf
	default:
		return log.Printf
	}
}

func printf(logLevel log.Level, message string) {
	logger := logfn(logLevel)
	logger(message)
}

func announcement(message string) {
	err := beeep.Notify("naisdevice", message, "../Resources/nais-logo-red.png")
	if err != nil {
		log.Errorf("sending notification: %s", err)
	}
}

func Infof(notify bool, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	printf(log.InfoLevel, message)
	if notify {
		announcement(message)
	}
}

func Errorf(notify bool, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	printf(log.ErrorLevel, message)
	if notify {
		announcement(message)
	}
}

func IsError(args ...any) bool {
	for _, arg := range args {
		if _, ok := arg.(error); ok {
			return true
		}
	}
	return false
}
