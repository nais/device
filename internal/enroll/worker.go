package enroll

import (
	"context"
)

type Worker interface {
	Run(ctx context.Context) error
	Send(ctx context.Context, req *DeviceRequest) (*Response, error)
}
