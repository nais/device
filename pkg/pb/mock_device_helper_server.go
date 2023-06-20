// Code generated by mockery v2.30.1. DO NOT EDIT.

package pb

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockDeviceHelperServer is an autogenerated mock type for the DeviceHelperServer type
type MockDeviceHelperServer struct {
	mock.Mock
}

// Configure provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceHelperServer) Configure(_a0 context.Context, _a1 *Configuration) (*ConfigureResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *ConfigureResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *Configuration) (*ConfigureResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *Configuration) *ConfigureResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ConfigureResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *Configuration) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSerial provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceHelperServer) GetSerial(_a0 context.Context, _a1 *GetSerialRequest) (*GetSerialResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *GetSerialResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetSerialRequest) (*GetSerialResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetSerialRequest) *GetSerialResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*GetSerialResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetSerialRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Teardown provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceHelperServer) Teardown(_a0 context.Context, _a1 *TeardownRequest) (*TeardownResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *TeardownResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *TeardownRequest) (*TeardownResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *TeardownRequest) *TeardownResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*TeardownResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *TeardownRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Upgrade provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceHelperServer) Upgrade(_a0 context.Context, _a1 *UpgradeRequest) (*UpgradeResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *UpgradeResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *UpgradeRequest) (*UpgradeResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *UpgradeRequest) *UpgradeResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*UpgradeResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *UpgradeRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mustEmbedUnimplementedDeviceHelperServer provides a mock function with given fields:
func (_m *MockDeviceHelperServer) mustEmbedUnimplementedDeviceHelperServer() {
	_m.Called()
}

// NewMockDeviceHelperServer creates a new instance of MockDeviceHelperServer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDeviceHelperServer(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDeviceHelperServer {
	mock := &MockDeviceHelperServer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
