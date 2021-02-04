package main

import (
	"fmt"

	"google.golang.org/grpc"

	"github.com/gen2brain/beeep"
	pb "github.com/nais/device/pkg/protobuf"
	log "github.com/sirupsen/logrus"
)

func onReady() {
	connection, err := grpc.Dial(
		fmt.Sprintf("127.0.0.1:%d", cfg.grpcPort),
		grpc.WithBlock(),
		grpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("unable to connect to naisdevice-agent grpc server: %v", err)
	}
	client := pb.NewDeviceAgentClient(connection)

	gui := NewGUI(client)
	if cfg.autoConnect {
		gui.Events <- ConnectClicked
	}

	go gui.handleButtonClicks()
	go gui.EventLoop()
	// TODO: go checkVersion(versionCheckInterval, gui)
}

func onExit() {
	// This is where we clean up
}

func notify(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	err := beeep.Notify("NAIS device", message, "../Resources/nais-logo-red.png")
	log.Infof("sending message to notification centre: %s", message)
	if err != nil {
		log.Errorf("failed sending message due to error: %s", err)
	}
}
