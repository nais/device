// Code generated by mockery. DO NOT EDIT.

package runtimeconfig

import (
	context "context"

	auth "github.com/nais/device/internal/device-agent/auth"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"

	pb "github.com/nais/device/internal/pb"
)

// MockRuntimeConfig is an autogenerated mock type for the RuntimeConfig type
type MockRuntimeConfig struct {
	mock.Mock
}

type MockRuntimeConfig_Expecter struct {
	mock *mock.Mock
}

func (_m *MockRuntimeConfig) EXPECT() *MockRuntimeConfig_Expecter {
	return &MockRuntimeConfig_Expecter{mock: &_m.Mock}
}

// APIServerPeer provides a mock function with given fields:
func (_m *MockRuntimeConfig) APIServerPeer() *pb.Gateway {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for APIServerPeer")
	}

	var r0 *pb.Gateway
	if rf, ok := ret.Get(0).(func() *pb.Gateway); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Gateway)
		}
	}

	return r0
}

// MockRuntimeConfig_APIServerPeer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'APIServerPeer'
type MockRuntimeConfig_APIServerPeer_Call struct {
	*mock.Call
}

// APIServerPeer is a helper method to define mock.On call
func (_e *MockRuntimeConfig_Expecter) APIServerPeer() *MockRuntimeConfig_APIServerPeer_Call {
	return &MockRuntimeConfig_APIServerPeer_Call{Call: _e.mock.On("APIServerPeer")}
}

func (_c *MockRuntimeConfig_APIServerPeer_Call) Run(run func()) *MockRuntimeConfig_APIServerPeer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRuntimeConfig_APIServerPeer_Call) Return(_a0 *pb.Gateway) *MockRuntimeConfig_APIServerPeer_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRuntimeConfig_APIServerPeer_Call) RunAndReturn(run func() *pb.Gateway) *MockRuntimeConfig_APIServerPeer_Call {
	_c.Call.Return(run)
	return _c
}

// BuildHelperConfiguration provides a mock function with given fields: peers
func (_m *MockRuntimeConfig) BuildHelperConfiguration(peers []*pb.Gateway) *pb.Configuration {
	ret := _m.Called(peers)

	if len(ret) == 0 {
		panic("no return value specified for BuildHelperConfiguration")
	}

	var r0 *pb.Configuration
	if rf, ok := ret.Get(0).(func([]*pb.Gateway) *pb.Configuration); ok {
		r0 = rf(peers)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Configuration)
		}
	}

	return r0
}

// MockRuntimeConfig_BuildHelperConfiguration_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'BuildHelperConfiguration'
type MockRuntimeConfig_BuildHelperConfiguration_Call struct {
	*mock.Call
}

// BuildHelperConfiguration is a helper method to define mock.On call
//   - peers []*pb.Gateway
func (_e *MockRuntimeConfig_Expecter) BuildHelperConfiguration(peers interface{}) *MockRuntimeConfig_BuildHelperConfiguration_Call {
	return &MockRuntimeConfig_BuildHelperConfiguration_Call{Call: _e.mock.On("BuildHelperConfiguration", peers)}
}

func (_c *MockRuntimeConfig_BuildHelperConfiguration_Call) Run(run func(peers []*pb.Gateway)) *MockRuntimeConfig_BuildHelperConfiguration_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]*pb.Gateway))
	})
	return _c
}

func (_c *MockRuntimeConfig_BuildHelperConfiguration_Call) Return(_a0 *pb.Configuration) *MockRuntimeConfig_BuildHelperConfiguration_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRuntimeConfig_BuildHelperConfiguration_Call) RunAndReturn(run func([]*pb.Gateway) *pb.Configuration) *MockRuntimeConfig_BuildHelperConfiguration_Call {
	_c.Call.Return(run)
	return _c
}

// ConnectToAPIServer provides a mock function with given fields: _a0
func (_m *MockRuntimeConfig) ConnectToAPIServer(_a0 context.Context) (pb.APIServerClient, func(), error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for ConnectToAPIServer")
	}

	var r0 pb.APIServerClient
	var r1 func()
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context) (pb.APIServerClient, func(), error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) pb.APIServerClient); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(pb.APIServerClient)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) func()); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(func())
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// MockRuntimeConfig_ConnectToAPIServer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ConnectToAPIServer'
type MockRuntimeConfig_ConnectToAPIServer_Call struct {
	*mock.Call
}

// ConnectToAPIServer is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *MockRuntimeConfig_Expecter) ConnectToAPIServer(_a0 interface{}) *MockRuntimeConfig_ConnectToAPIServer_Call {
	return &MockRuntimeConfig_ConnectToAPIServer_Call{Call: _e.mock.On("ConnectToAPIServer", _a0)}
}

func (_c *MockRuntimeConfig_ConnectToAPIServer_Call) Run(run func(_a0 context.Context)) *MockRuntimeConfig_ConnectToAPIServer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockRuntimeConfig_ConnectToAPIServer_Call) Return(_a0 pb.APIServerClient, _a1 func(), _a2 error) *MockRuntimeConfig_ConnectToAPIServer_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *MockRuntimeConfig_ConnectToAPIServer_Call) RunAndReturn(run func(context.Context) (pb.APIServerClient, func(), error)) *MockRuntimeConfig_ConnectToAPIServer_Call {
	_c.Call.Return(run)
	return _c
}

// DialAPIServer provides a mock function with given fields: _a0
func (_m *MockRuntimeConfig) DialAPIServer(_a0 context.Context) (*grpc.ClientConn, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for DialAPIServer")
	}

	var r0 *grpc.ClientConn
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*grpc.ClientConn, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *grpc.ClientConn); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*grpc.ClientConn)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockRuntimeConfig_DialAPIServer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DialAPIServer'
type MockRuntimeConfig_DialAPIServer_Call struct {
	*mock.Call
}

// DialAPIServer is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *MockRuntimeConfig_Expecter) DialAPIServer(_a0 interface{}) *MockRuntimeConfig_DialAPIServer_Call {
	return &MockRuntimeConfig_DialAPIServer_Call{Call: _e.mock.On("DialAPIServer", _a0)}
}

func (_c *MockRuntimeConfig_DialAPIServer_Call) Run(run func(_a0 context.Context)) *MockRuntimeConfig_DialAPIServer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockRuntimeConfig_DialAPIServer_Call) Return(_a0 *grpc.ClientConn, _a1 error) *MockRuntimeConfig_DialAPIServer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockRuntimeConfig_DialAPIServer_Call) RunAndReturn(run func(context.Context) (*grpc.ClientConn, error)) *MockRuntimeConfig_DialAPIServer_Call {
	_c.Call.Return(run)
	return _c
}

// EnsureEnrolled provides a mock function with given fields: _a0, _a1
func (_m *MockRuntimeConfig) EnsureEnrolled(_a0 context.Context, _a1 string) error {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for EnsureEnrolled")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockRuntimeConfig_EnsureEnrolled_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'EnsureEnrolled'
type MockRuntimeConfig_EnsureEnrolled_Call struct {
	*mock.Call
}

// EnsureEnrolled is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
func (_e *MockRuntimeConfig_Expecter) EnsureEnrolled(_a0 interface{}, _a1 interface{}) *MockRuntimeConfig_EnsureEnrolled_Call {
	return &MockRuntimeConfig_EnsureEnrolled_Call{Call: _e.mock.On("EnsureEnrolled", _a0, _a1)}
}

func (_c *MockRuntimeConfig_EnsureEnrolled_Call) Run(run func(_a0 context.Context, _a1 string)) *MockRuntimeConfig_EnsureEnrolled_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockRuntimeConfig_EnsureEnrolled_Call) Return(_a0 error) *MockRuntimeConfig_EnsureEnrolled_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRuntimeConfig_EnsureEnrolled_Call) RunAndReturn(run func(context.Context, string) error) *MockRuntimeConfig_EnsureEnrolled_Call {
	_c.Call.Return(run)
	return _c
}

// GetActiveTenant provides a mock function with given fields:
func (_m *MockRuntimeConfig) GetActiveTenant() *pb.Tenant {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetActiveTenant")
	}

	var r0 *pb.Tenant
	if rf, ok := ret.Get(0).(func() *pb.Tenant); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Tenant)
		}
	}

	return r0
}

// MockRuntimeConfig_GetActiveTenant_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetActiveTenant'
type MockRuntimeConfig_GetActiveTenant_Call struct {
	*mock.Call
}

// GetActiveTenant is a helper method to define mock.On call
func (_e *MockRuntimeConfig_Expecter) GetActiveTenant() *MockRuntimeConfig_GetActiveTenant_Call {
	return &MockRuntimeConfig_GetActiveTenant_Call{Call: _e.mock.On("GetActiveTenant")}
}

func (_c *MockRuntimeConfig_GetActiveTenant_Call) Run(run func()) *MockRuntimeConfig_GetActiveTenant_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRuntimeConfig_GetActiveTenant_Call) Return(_a0 *pb.Tenant) *MockRuntimeConfig_GetActiveTenant_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRuntimeConfig_GetActiveTenant_Call) RunAndReturn(run func() *pb.Tenant) *MockRuntimeConfig_GetActiveTenant_Call {
	_c.Call.Return(run)
	return _c
}

// GetTenantSession provides a mock function with given fields:
func (_m *MockRuntimeConfig) GetTenantSession() (*pb.Session, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetTenantSession")
	}

	var r0 *pb.Session
	var r1 error
	if rf, ok := ret.Get(0).(func() (*pb.Session, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *pb.Session); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Session)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockRuntimeConfig_GetTenantSession_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetTenantSession'
type MockRuntimeConfig_GetTenantSession_Call struct {
	*mock.Call
}

// GetTenantSession is a helper method to define mock.On call
func (_e *MockRuntimeConfig_Expecter) GetTenantSession() *MockRuntimeConfig_GetTenantSession_Call {
	return &MockRuntimeConfig_GetTenantSession_Call{Call: _e.mock.On("GetTenantSession")}
}

func (_c *MockRuntimeConfig_GetTenantSession_Call) Run(run func()) *MockRuntimeConfig_GetTenantSession_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRuntimeConfig_GetTenantSession_Call) Return(_a0 *pb.Session, _a1 error) *MockRuntimeConfig_GetTenantSession_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockRuntimeConfig_GetTenantSession_Call) RunAndReturn(run func() (*pb.Session, error)) *MockRuntimeConfig_GetTenantSession_Call {
	_c.Call.Return(run)
	return _c
}

// GetToken provides a mock function with given fields: _a0
func (_m *MockRuntimeConfig) GetToken(_a0 context.Context) (string, error) {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for GetToken")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (string, error)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(context.Context) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockRuntimeConfig_GetToken_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetToken'
type MockRuntimeConfig_GetToken_Call struct {
	*mock.Call
}

// GetToken is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *MockRuntimeConfig_Expecter) GetToken(_a0 interface{}) *MockRuntimeConfig_GetToken_Call {
	return &MockRuntimeConfig_GetToken_Call{Call: _e.mock.On("GetToken", _a0)}
}

func (_c *MockRuntimeConfig_GetToken_Call) Run(run func(_a0 context.Context)) *MockRuntimeConfig_GetToken_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockRuntimeConfig_GetToken_Call) Return(_a0 string, _a1 error) *MockRuntimeConfig_GetToken_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockRuntimeConfig_GetToken_Call) RunAndReturn(run func(context.Context) (string, error)) *MockRuntimeConfig_GetToken_Call {
	_c.Call.Return(run)
	return _c
}

// LoadEnrollConfig provides a mock function with given fields:
func (_m *MockRuntimeConfig) LoadEnrollConfig() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for LoadEnrollConfig")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockRuntimeConfig_LoadEnrollConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'LoadEnrollConfig'
type MockRuntimeConfig_LoadEnrollConfig_Call struct {
	*mock.Call
}

// LoadEnrollConfig is a helper method to define mock.On call
func (_e *MockRuntimeConfig_Expecter) LoadEnrollConfig() *MockRuntimeConfig_LoadEnrollConfig_Call {
	return &MockRuntimeConfig_LoadEnrollConfig_Call{Call: _e.mock.On("LoadEnrollConfig")}
}

func (_c *MockRuntimeConfig_LoadEnrollConfig_Call) Run(run func()) *MockRuntimeConfig_LoadEnrollConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRuntimeConfig_LoadEnrollConfig_Call) Return(_a0 error) *MockRuntimeConfig_LoadEnrollConfig_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRuntimeConfig_LoadEnrollConfig_Call) RunAndReturn(run func() error) *MockRuntimeConfig_LoadEnrollConfig_Call {
	_c.Call.Return(run)
	return _c
}

// PopulateTenants provides a mock function with given fields: _a0
func (_m *MockRuntimeConfig) PopulateTenants(_a0 context.Context) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for PopulateTenants")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockRuntimeConfig_PopulateTenants_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PopulateTenants'
type MockRuntimeConfig_PopulateTenants_Call struct {
	*mock.Call
}

// PopulateTenants is a helper method to define mock.On call
//   - _a0 context.Context
func (_e *MockRuntimeConfig_Expecter) PopulateTenants(_a0 interface{}) *MockRuntimeConfig_PopulateTenants_Call {
	return &MockRuntimeConfig_PopulateTenants_Call{Call: _e.mock.On("PopulateTenants", _a0)}
}

func (_c *MockRuntimeConfig_PopulateTenants_Call) Run(run func(_a0 context.Context)) *MockRuntimeConfig_PopulateTenants_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockRuntimeConfig_PopulateTenants_Call) Return(_a0 error) *MockRuntimeConfig_PopulateTenants_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRuntimeConfig_PopulateTenants_Call) RunAndReturn(run func(context.Context) error) *MockRuntimeConfig_PopulateTenants_Call {
	_c.Call.Return(run)
	return _c
}

// ResetEnrollConfig provides a mock function with given fields:
func (_m *MockRuntimeConfig) ResetEnrollConfig() {
	_m.Called()
}

// MockRuntimeConfig_ResetEnrollConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ResetEnrollConfig'
type MockRuntimeConfig_ResetEnrollConfig_Call struct {
	*mock.Call
}

// ResetEnrollConfig is a helper method to define mock.On call
func (_e *MockRuntimeConfig_Expecter) ResetEnrollConfig() *MockRuntimeConfig_ResetEnrollConfig_Call {
	return &MockRuntimeConfig_ResetEnrollConfig_Call{Call: _e.mock.On("ResetEnrollConfig")}
}

func (_c *MockRuntimeConfig_ResetEnrollConfig_Call) Run(run func()) *MockRuntimeConfig_ResetEnrollConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRuntimeConfig_ResetEnrollConfig_Call) Return() *MockRuntimeConfig_ResetEnrollConfig_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockRuntimeConfig_ResetEnrollConfig_Call) RunAndReturn(run func()) *MockRuntimeConfig_ResetEnrollConfig_Call {
	_c.Call.Return(run)
	return _c
}

// SaveEnrollConfig provides a mock function with given fields:
func (_m *MockRuntimeConfig) SaveEnrollConfig() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for SaveEnrollConfig")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockRuntimeConfig_SaveEnrollConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SaveEnrollConfig'
type MockRuntimeConfig_SaveEnrollConfig_Call struct {
	*mock.Call
}

// SaveEnrollConfig is a helper method to define mock.On call
func (_e *MockRuntimeConfig_Expecter) SaveEnrollConfig() *MockRuntimeConfig_SaveEnrollConfig_Call {
	return &MockRuntimeConfig_SaveEnrollConfig_Call{Call: _e.mock.On("SaveEnrollConfig")}
}

func (_c *MockRuntimeConfig_SaveEnrollConfig_Call) Run(run func()) *MockRuntimeConfig_SaveEnrollConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRuntimeConfig_SaveEnrollConfig_Call) Return(_a0 error) *MockRuntimeConfig_SaveEnrollConfig_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRuntimeConfig_SaveEnrollConfig_Call) RunAndReturn(run func() error) *MockRuntimeConfig_SaveEnrollConfig_Call {
	_c.Call.Return(run)
	return _c
}

// SetActiveTenant provides a mock function with given fields: _a0
func (_m *MockRuntimeConfig) SetActiveTenant(_a0 string) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for SetActiveTenant")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockRuntimeConfig_SetActiveTenant_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetActiveTenant'
type MockRuntimeConfig_SetActiveTenant_Call struct {
	*mock.Call
}

// SetActiveTenant is a helper method to define mock.On call
//   - _a0 string
func (_e *MockRuntimeConfig_Expecter) SetActiveTenant(_a0 interface{}) *MockRuntimeConfig_SetActiveTenant_Call {
	return &MockRuntimeConfig_SetActiveTenant_Call{Call: _e.mock.On("SetActiveTenant", _a0)}
}

func (_c *MockRuntimeConfig_SetActiveTenant_Call) Run(run func(_a0 string)) *MockRuntimeConfig_SetActiveTenant_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *MockRuntimeConfig_SetActiveTenant_Call) Return(_a0 error) *MockRuntimeConfig_SetActiveTenant_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRuntimeConfig_SetActiveTenant_Call) RunAndReturn(run func(string) error) *MockRuntimeConfig_SetActiveTenant_Call {
	_c.Call.Return(run)
	return _c
}

// SetTenantSession provides a mock function with given fields: _a0
func (_m *MockRuntimeConfig) SetTenantSession(_a0 *pb.Session) error {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for SetTenantSession")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*pb.Session) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockRuntimeConfig_SetTenantSession_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetTenantSession'
type MockRuntimeConfig_SetTenantSession_Call struct {
	*mock.Call
}

// SetTenantSession is a helper method to define mock.On call
//   - _a0 *pb.Session
func (_e *MockRuntimeConfig_Expecter) SetTenantSession(_a0 interface{}) *MockRuntimeConfig_SetTenantSession_Call {
	return &MockRuntimeConfig_SetTenantSession_Call{Call: _e.mock.On("SetTenantSession", _a0)}
}

func (_c *MockRuntimeConfig_SetTenantSession_Call) Run(run func(_a0 *pb.Session)) *MockRuntimeConfig_SetTenantSession_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*pb.Session))
	})
	return _c
}

func (_c *MockRuntimeConfig_SetTenantSession_Call) Return(_a0 error) *MockRuntimeConfig_SetTenantSession_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRuntimeConfig_SetTenantSession_Call) RunAndReturn(run func(*pb.Session) error) *MockRuntimeConfig_SetTenantSession_Call {
	_c.Call.Return(run)
	return _c
}

// SetToken provides a mock function with given fields: _a0
func (_m *MockRuntimeConfig) SetToken(_a0 *auth.Tokens) {
	_m.Called(_a0)
}

// MockRuntimeConfig_SetToken_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetToken'
type MockRuntimeConfig_SetToken_Call struct {
	*mock.Call
}

// SetToken is a helper method to define mock.On call
//   - _a0 *auth.Tokens
func (_e *MockRuntimeConfig_Expecter) SetToken(_a0 interface{}) *MockRuntimeConfig_SetToken_Call {
	return &MockRuntimeConfig_SetToken_Call{Call: _e.mock.On("SetToken", _a0)}
}

func (_c *MockRuntimeConfig_SetToken_Call) Run(run func(_a0 *auth.Tokens)) *MockRuntimeConfig_SetToken_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*auth.Tokens))
	})
	return _c
}

func (_c *MockRuntimeConfig_SetToken_Call) Return() *MockRuntimeConfig_SetToken_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockRuntimeConfig_SetToken_Call) RunAndReturn(run func(*auth.Tokens)) *MockRuntimeConfig_SetToken_Call {
	_c.Call.Return(run)
	return _c
}

// Tenants provides a mock function with given fields:
func (_m *MockRuntimeConfig) Tenants() []*pb.Tenant {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Tenants")
	}

	var r0 []*pb.Tenant
	if rf, ok := ret.Get(0).(func() []*pb.Tenant); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pb.Tenant)
		}
	}

	return r0
}

// MockRuntimeConfig_Tenants_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Tenants'
type MockRuntimeConfig_Tenants_Call struct {
	*mock.Call
}

// Tenants is a helper method to define mock.On call
func (_e *MockRuntimeConfig_Expecter) Tenants() *MockRuntimeConfig_Tenants_Call {
	return &MockRuntimeConfig_Tenants_Call{Call: _e.mock.On("Tenants")}
}

func (_c *MockRuntimeConfig_Tenants_Call) Run(run func()) *MockRuntimeConfig_Tenants_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRuntimeConfig_Tenants_Call) Return(_a0 []*pb.Tenant) *MockRuntimeConfig_Tenants_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRuntimeConfig_Tenants_Call) RunAndReturn(run func() []*pb.Tenant) *MockRuntimeConfig_Tenants_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockRuntimeConfig creates a new instance of MockRuntimeConfig. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockRuntimeConfig(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockRuntimeConfig {
	mock := &MockRuntimeConfig{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
