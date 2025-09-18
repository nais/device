package enroll

import (
	"context"

	"github.com/sirupsen/logrus"
)

type noopWorker struct {
	log *logrus.Entry
}

func NewNoopWorker(ctx context.Context, log *logrus.Entry) Worker {
	return &noopWorker{
		log: log,
	}
}

func (w *noopWorker) Run(ctx context.Context) error {
	w.log.Info("noop worker: running")
	return nil
}

func (w *noopWorker) Send(ctx context.Context, req *DeviceRequest) (*Response, error) {
	w.log.WithField("request", req).Info("noop worker")
	return &Response{
		APIServerGRPCAddress: "1.1.1.1:9077",
		WireGuardIPv4:        "1.2.3.4",
		WireGuardIPv6:        "fd00::1",
	}, nil
}
