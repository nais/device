package kolide

import (
	"context"
)

type FakeClient struct {
	Devices []Device
}

// GetChecks implements Client.
func (f *FakeClient) GetChecks(ctx context.Context) ([]*Check, error) {
	panic("unimplemented")
}

// GetDeviceIssues implements Client.
func (f *FakeClient) GetDeviceIssues(ctx context.Context, deviceID string) ([]*DeviceFailure, error) {
	panic("unimplemented")
}

// GetDevices implements Client.
func (f *FakeClient) GetDevices(ctx context.Context) ([]*Device, error) {
	panic("unimplemented")
}

// GetIssues implements Client.
func (f *FakeClient) GetIssues(ctx context.Context) ([]*DeviceFailure, error) {
	panic("unimplemented")
}

var _ Client = &FakeClient{}

func (f *FakeClient) Build() Client {
	return f
}
