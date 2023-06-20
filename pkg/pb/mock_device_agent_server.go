// Code generated by mockery v2.30.1. DO NOT EDIT.

package pb

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockDeviceAgentServer is an autogenerated mock type for the DeviceAgentServer type
type MockDeviceAgentServer struct {
	mock.Mock
}

// ConfigureJITA provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceAgentServer) ConfigureJITA(_a0 context.Context, _a1 *ConfigureJITARequest) (*ConfigureJITAResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *ConfigureJITAResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *ConfigureJITARequest) (*ConfigureJITAResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *ConfigureJITARequest) *ConfigureJITAResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ConfigureJITAResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *ConfigureJITARequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAgentConfiguration provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceAgentServer) GetAgentConfiguration(_a0 context.Context, _a1 *GetAgentConfigurationRequest) (*GetAgentConfigurationResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *GetAgentConfigurationResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *GetAgentConfigurationRequest) (*GetAgentConfigurationResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *GetAgentConfigurationRequest) *GetAgentConfigurationResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*GetAgentConfigurationResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *GetAgentConfigurationRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Login provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceAgentServer) Login(_a0 context.Context, _a1 *LoginRequest) (*LoginResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *LoginResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *LoginRequest) (*LoginResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *LoginRequest) *LoginResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*LoginResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *LoginRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Logout provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceAgentServer) Logout(_a0 context.Context, _a1 *LogoutRequest) (*LogoutResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *LogoutResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *LogoutRequest) (*LogoutResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *LogoutRequest) *LogoutResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*LogoutResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *LogoutRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetActiveTenant provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceAgentServer) SetActiveTenant(_a0 context.Context, _a1 *SetActiveTenantRequest) (*SetActiveTenantResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *SetActiveTenantResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *SetActiveTenantRequest) (*SetActiveTenantResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *SetActiveTenantRequest) *SetActiveTenantResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*SetActiveTenantResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *SetActiveTenantRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetAgentConfiguration provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceAgentServer) SetAgentConfiguration(_a0 context.Context, _a1 *SetAgentConfigurationRequest) (*SetAgentConfigurationResponse, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *SetAgentConfigurationResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *SetAgentConfigurationRequest) (*SetAgentConfigurationResponse, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *SetAgentConfigurationRequest) *SetAgentConfigurationResponse); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*SetAgentConfigurationResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *SetAgentConfigurationRequest) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Status provides a mock function with given fields: _a0, _a1
func (_m *MockDeviceAgentServer) Status(_a0 *AgentStatusRequest, _a1 DeviceAgent_StatusServer) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*AgentStatusRequest, DeviceAgent_StatusServer) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mustEmbedUnimplementedDeviceAgentServer provides a mock function with given fields:
func (_m *MockDeviceAgentServer) mustEmbedUnimplementedDeviceAgentServer() {
	_m.Called()
}

// NewMockDeviceAgentServer creates a new instance of MockDeviceAgentServer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDeviceAgentServer(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDeviceAgentServer {
	mock := &MockDeviceAgentServer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
