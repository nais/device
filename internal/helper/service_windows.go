package helper

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/svc"
)

const serviceName = "naisdevice-agent-helper"

type MyService struct {
	programContext context.Context
	cancel         context.CancelFunc
	log            *logrus.Entry
}

func StartService(log *logrus.Entry, programContext context.Context, cancel context.CancelFunc) error {
	isWindowsService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}

	if !isWindowsService {
		return nil
	}

	go func() {
		s := &MyService{
			programContext: programContext,
			cancel:         cancel,
			log:            log,
		}

		err = svc.Run(serviceName, s)
		if err != nil {
			log.Fatalf("Running service: %v", err)
		}
	}()

	log.Infof("ran service handler")
	return nil
}

func (service *MyService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	service.log.Infof("service started with args: %v", args)
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case <-service.programContext.Done():
			break loop
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				service.log.Infof("Stop service: %v", c)
				service.cancel()
				break loop
			default:
				service.log.Errorf("unexpected control request #%d", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
