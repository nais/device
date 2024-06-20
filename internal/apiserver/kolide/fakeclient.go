package kolide

import (
	"context"

	"github.com/nais/device/internal/pb"
)

type FakeClient struct {
	Devices []Device
}

var _ Client = &FakeClient{}

// FillKolideData implements Client.
func (f *FakeClient) FillKolideData(ctx context.Context, devices []*pb.Device) error {
	panic("unimplemented")
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
