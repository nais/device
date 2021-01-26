package main

import (
	"fmt"
	"github.com/nais/device/device-agent/open"
	"github.com/nais/device/pkg/logger"

	"github.com/gen2brain/beeep"
	log "github.com/sirupsen/logrus"
)


func onReady() {
	gui := NewGUI()
	if cfg.autoConnect {
		gui.Events <- ConnectClicked
	}

	go gui.EventLoop()
	//TODO: go checkVersion(versionCheckInterval, gui)
}

func handleGuiEvent(guiEvent GuiEvent, state GuiState) {
	switch guiEvent {
	case VersionClicked:
		err := open.Open(softwareReleasePage)
		if err != nil {
			log.Warn("opening latest release url: %w", err)
		}

	case StateInfoClicked:
		err := open.Open(slackURL)
		if err != nil {
			log.Warnf("opening slack: %v", err)
		}

	case ConnectClicked:
		log.Infof("Connect button clicked")
		if state == GuiStateDisconnected {
			// TODO: RPC call to initiate connection
		} else {
			// TODO: RPC call to disconnect current connection
		}

	case HelperLogClicked:
		err := open.Open(logger.DeviceAgentHelperLogFilePath())
		if err != nil {
			log.Warn("opening device agent helper log: %w", err)
		}

	case DeviceLogClicked:
		err := open.Open(logger.DeviceAgentLogFilePath(cfg.configDir))
		if err != nil {
			log.Warn("opening device agent log: %w", err)
		}

	case QuitClicked:
		// TODO: RPC call to quit device-agent
	}
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
