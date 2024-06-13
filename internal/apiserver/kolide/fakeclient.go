package kolide

import (
	"context"

	"github.com/nais/device/internal/pb"
)

type FakeClient struct{}

var _ Client = &FakeClient{}

// NewFakeClient returns a fake client that returns the provided devices.
func NewFakeClient() *FakeClient {
	return &FakeClient{}
}

func (f *FakeClient) Build() Client {
	return f
}

// RefreshCache implements Client.
func (f *FakeClient) RefreshCache(ctx context.Context) error {
	// no-op
	return nil
}

// GetDeviceFailures implements Client.
func (f *FakeClient) GetDeviceFailures(ctx context.Context, deviceID string) ([]*pb.DeviceIssue, error) {
	panic("unimplemented")
}

// DumpChecks implements Client.
func (f *FakeClient) DumpChecks() ([]byte, error) {
	panic("unimplemented")
}

// UpdateDeviceFailures implements Client.
func (f *FakeClient) UpdateDeviceFailures(ctx context.Context) error {
	panic("unimplemented")
}
