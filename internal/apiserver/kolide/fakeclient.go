package kolide

import (
	"context"
	"fmt"
	"strings"
)

type FakeClient struct {
	devices []Device
}

var _ Client = &FakeClient{}

// NewFakeClient returns a fake client that returns the provided devices.
func NewFakeClient() *FakeClient {
	return &FakeClient{}
}

func (f *FakeClient) WithDevice(device Device) *FakeClient {
	return &FakeClient{
		devices: append(f.devices, device),
	}
}

func (f *FakeClient) Build() Client {
	return f
}

// GetDevice implements Client.
func (f *FakeClient) GetDevice(ctx context.Context, email, platform, serial string) (Device, error) {
	for _, d := range f.devices {
		if strings.EqualFold(d.AssignedOwner.Email, email) &&
			strings.EqualFold(d.Platform, platform) &&
			strings.EqualFold(d.Serial, serial) {
			return d, nil
		}
	}

	return Device{}, fmt.Errorf("device (%v, %v, %v) not found in fake client. use WithDevice() before Build() to add it. we currently have: %+v", email, platform, serial, f.devices)
}

// RefreshCache implements Client.
func (f *FakeClient) RefreshCache(ctx context.Context) error {
	// no-op
	return nil
}
