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

// GetDevice implements Client.
func (f *FakeClient) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	panic("unimplemented")
}

// GetDeviceIssues implements Client.
func (f *FakeClient) GetDeviceIssues(ctx context.Context, deviceID string) ([]*Issue, error) {
	panic("unimplemented")
}

// GetDevices implements Client.
func (f *FakeClient) GetDevices(ctx context.Context) ([]*Device, error) {
	panic("unimplemented")
}

// GetIssues implements Client.
func (f *FakeClient) GetIssues(ctx context.Context) ([]*Issue, error) {
	panic("unimplemented")
}

// GetPeople implements Client.
func (f *FakeClient) GetPeople(ctx context.Context) (map[string]*Person, error) {
	panic("unimplemented")
}

var _ Client = &FakeClient{}

func (f *FakeClient) Build() Client {
	return f
}
