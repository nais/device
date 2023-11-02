package systray

import (
	"context"

	"github.com/getlantern/systray"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/nais/device/pkg/notify"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
)

const ConfigFile = "systray-config.json"

type trayState struct {
	ctx        context.Context
	log        *logrus.Entry
	cfg        Config
	notifier   notify.Notifier
	connection *grpc.ClientConn
}

func (s *trayState) onReady() {
	var err error

	s.connection, err = grpc.Dial(
		"unix:"+s.cfg.GrpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		s.log.Fatalf("unable to connect to naisdevice-agent grpc server: %v", err)
	}

	client := pb.NewDeviceAgentClient(s.connection)

	gui := NewGUI(s.ctx, s.log, client, s.cfg, s.notifier)

	go gui.handleStatusStream(s.ctx)
	go gui.handleButtonClicks(s.ctx)
	go gui.EventLoop(s.ctx)
}

func (s *trayState) onExit() {
	if s.connection != nil {
		s.connection.Close()
	}
}

func Spawn(ctx context.Context, log *logrus.Entry, cfg Config, notifier notify.Notifier) {
	state := trayState{
		ctx:      ctx,
		log:      log,
		cfg:      cfg,
		notifier: notifier,
	}
	systray.Run(state.onReady, state.onExit)
}
