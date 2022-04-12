package outtune

import (
	"context"

	"github.com/nais/device/pkg/pb"
)

type windows struct{}

func New(_ pb.DeviceHelperClient) Outtune {
	return &windows{}
}

func (o *windows) Install(ctx context.Context) error {
	return nil
}

func (o *windows) Cleanup(ctx context.Context) error {
	return nil
}
