// Code generated by mockery v2.32.2. DO NOT EDIT.

package wireguard

import (
	io "io"

	mock "github.com/stretchr/testify/mock"
)

// MockWireGuardPeerConfig is an autogenerated mock type for the WireGuardPeerConfig type
type MockWireGuardPeerConfig struct {
	mock.Mock
}

// GetTunnelIP provides a mock function with given fields:
func (_m *MockWireGuardPeerConfig) GetTunnelIP() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetWireGuardConfigPath provides a mock function with given fields:
func (_m *MockWireGuardPeerConfig) GetWireGuardConfigPath() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// WriteWireGuardBase provides a mock function with given fields: _a0
func (_m *MockWireGuardPeerConfig) WriteWireGuardBase(_a0 io.Writer) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(io.Writer) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockWireGuardPeerConfig creates a new instance of MockWireGuardPeerConfig. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockWireGuardPeerConfig(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockWireGuardPeerConfig {
	mock := &MockWireGuardPeerConfig{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
