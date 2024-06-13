// Code generated by mockery. DO NOT EDIT.

package database

import (
	context "context"

	pb "github.com/nais/device/internal/pb"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// MockAPIServer is an autogenerated mock type for the APIServer type
type MockAPIServer struct {
	mock.Mock
}

type MockAPIServer_Expecter struct {
	mock *mock.Mock
}

func (_m *MockAPIServer) EXPECT() *MockAPIServer_Expecter {
	return &MockAPIServer_Expecter{mock: &_m.Mock}
}

// AddDevice provides a mock function with given fields: ctx, device
func (_m *MockAPIServer) AddDevice(ctx context.Context, device *pb.Device) error {
	ret := _m.Called(ctx, device)

	if len(ret) == 0 {
		panic("no return value specified for AddDevice")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *pb.Device) error); ok {
		r0 = rf(ctx, device)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAPIServer_AddDevice_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddDevice'
type MockAPIServer_AddDevice_Call struct {
	*mock.Call
}

// AddDevice is a helper method to define mock.On call
//   - ctx context.Context
//   - device *pb.Device
func (_e *MockAPIServer_Expecter) AddDevice(ctx interface{}, device interface{}) *MockAPIServer_AddDevice_Call {
	return &MockAPIServer_AddDevice_Call{Call: _e.mock.On("AddDevice", ctx, device)}
}

func (_c *MockAPIServer_AddDevice_Call) Run(run func(ctx context.Context, device *pb.Device)) *MockAPIServer_AddDevice_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*pb.Device))
	})
	return _c
}

func (_c *MockAPIServer_AddDevice_Call) Return(_a0 error) *MockAPIServer_AddDevice_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAPIServer_AddDevice_Call) RunAndReturn(run func(context.Context, *pb.Device) error) *MockAPIServer_AddDevice_Call {
	_c.Call.Return(run)
	return _c
}

// AddGateway provides a mock function with given fields: ctx, gateway
func (_m *MockAPIServer) AddGateway(ctx context.Context, gateway *pb.Gateway) error {
	ret := _m.Called(ctx, gateway)

	if len(ret) == 0 {
		panic("no return value specified for AddGateway")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *pb.Gateway) error); ok {
		r0 = rf(ctx, gateway)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAPIServer_AddGateway_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddGateway'
type MockAPIServer_AddGateway_Call struct {
	*mock.Call
}

// AddGateway is a helper method to define mock.On call
//   - ctx context.Context
//   - gateway *pb.Gateway
func (_e *MockAPIServer_Expecter) AddGateway(ctx interface{}, gateway interface{}) *MockAPIServer_AddGateway_Call {
	return &MockAPIServer_AddGateway_Call{Call: _e.mock.On("AddGateway", ctx, gateway)}
}

func (_c *MockAPIServer_AddGateway_Call) Run(run func(ctx context.Context, gateway *pb.Gateway)) *MockAPIServer_AddGateway_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*pb.Gateway))
	})
	return _c
}

func (_c *MockAPIServer_AddGateway_Call) Return(_a0 error) *MockAPIServer_AddGateway_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAPIServer_AddGateway_Call) RunAndReturn(run func(context.Context, *pb.Gateway) error) *MockAPIServer_AddGateway_Call {
	_c.Call.Return(run)
	return _c
}

// AddSessionInfo provides a mock function with given fields: ctx, si
func (_m *MockAPIServer) AddSessionInfo(ctx context.Context, si *pb.Session) error {
	ret := _m.Called(ctx, si)

	if len(ret) == 0 {
		panic("no return value specified for AddSessionInfo")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *pb.Session) error); ok {
		r0 = rf(ctx, si)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAPIServer_AddSessionInfo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddSessionInfo'
type MockAPIServer_AddSessionInfo_Call struct {
	*mock.Call
}

// AddSessionInfo is a helper method to define mock.On call
//   - ctx context.Context
//   - si *pb.Session
func (_e *MockAPIServer_Expecter) AddSessionInfo(ctx interface{}, si interface{}) *MockAPIServer_AddSessionInfo_Call {
	return &MockAPIServer_AddSessionInfo_Call{Call: _e.mock.On("AddSessionInfo", ctx, si)}
}

func (_c *MockAPIServer_AddSessionInfo_Call) Run(run func(ctx context.Context, si *pb.Session)) *MockAPIServer_AddSessionInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*pb.Session))
	})
	return _c
}

func (_c *MockAPIServer_AddSessionInfo_Call) Return(_a0 error) *MockAPIServer_AddSessionInfo_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAPIServer_AddSessionInfo_Call) RunAndReturn(run func(context.Context, *pb.Session) error) *MockAPIServer_AddSessionInfo_Call {
	_c.Call.Return(run)
	return _c
}

// ClearDeviceIssuesExceptFor provides a mock function with given fields: ctx, deviceIds
func (_m *MockAPIServer) ClearDeviceIssuesExceptFor(ctx context.Context, deviceIds []int64) error {
	ret := _m.Called(ctx, deviceIds)

	if len(ret) == 0 {
		panic("no return value specified for ClearDeviceIssuesExceptFor")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []int64) error); ok {
		r0 = rf(ctx, deviceIds)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAPIServer_ClearDeviceIssuesExceptFor_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ClearDeviceIssuesExceptFor'
type MockAPIServer_ClearDeviceIssuesExceptFor_Call struct {
	*mock.Call
}

// ClearDeviceIssuesExceptFor is a helper method to define mock.On call
//   - ctx context.Context
//   - deviceIds []int64
func (_e *MockAPIServer_Expecter) ClearDeviceIssuesExceptFor(ctx interface{}, deviceIds interface{}) *MockAPIServer_ClearDeviceIssuesExceptFor_Call {
	return &MockAPIServer_ClearDeviceIssuesExceptFor_Call{Call: _e.mock.On("ClearDeviceIssuesExceptFor", ctx, deviceIds)}
}

func (_c *MockAPIServer_ClearDeviceIssuesExceptFor_Call) Run(run func(ctx context.Context, deviceIds []int64)) *MockAPIServer_ClearDeviceIssuesExceptFor_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]int64))
	})
	return _c
}

func (_c *MockAPIServer_ClearDeviceIssuesExceptFor_Call) Return(_a0 error) *MockAPIServer_ClearDeviceIssuesExceptFor_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAPIServer_ClearDeviceIssuesExceptFor_Call) RunAndReturn(run func(context.Context, []int64) error) *MockAPIServer_ClearDeviceIssuesExceptFor_Call {
	_c.Call.Return(run)
	return _c
}

// ReadDevice provides a mock function with given fields: ctx, publicKey
func (_m *MockAPIServer) ReadDevice(ctx context.Context, publicKey string) (*pb.Device, error) {
	ret := _m.Called(ctx, publicKey)

	if len(ret) == 0 {
		panic("no return value specified for ReadDevice")
	}

	var r0 *pb.Device
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*pb.Device, error)); ok {
		return rf(ctx, publicKey)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *pb.Device); ok {
		r0 = rf(ctx, publicKey)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Device)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, publicKey)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAPIServer_ReadDevice_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReadDevice'
type MockAPIServer_ReadDevice_Call struct {
	*mock.Call
}

// ReadDevice is a helper method to define mock.On call
//   - ctx context.Context
//   - publicKey string
func (_e *MockAPIServer_Expecter) ReadDevice(ctx interface{}, publicKey interface{}) *MockAPIServer_ReadDevice_Call {
	return &MockAPIServer_ReadDevice_Call{Call: _e.mock.On("ReadDevice", ctx, publicKey)}
}

func (_c *MockAPIServer_ReadDevice_Call) Run(run func(ctx context.Context, publicKey string)) *MockAPIServer_ReadDevice_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockAPIServer_ReadDevice_Call) Return(_a0 *pb.Device, _a1 error) *MockAPIServer_ReadDevice_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAPIServer_ReadDevice_Call) RunAndReturn(run func(context.Context, string) (*pb.Device, error)) *MockAPIServer_ReadDevice_Call {
	_c.Call.Return(run)
	return _c
}

// ReadDeviceById provides a mock function with given fields: ctx, deviceID
func (_m *MockAPIServer) ReadDeviceById(ctx context.Context, deviceID int64) (*pb.Device, error) {
	ret := _m.Called(ctx, deviceID)

	if len(ret) == 0 {
		panic("no return value specified for ReadDeviceById")
	}

	var r0 *pb.Device
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) (*pb.Device, error)); ok {
		return rf(ctx, deviceID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int64) *pb.Device); ok {
		r0 = rf(ctx, deviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Device)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(ctx, deviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAPIServer_ReadDeviceById_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReadDeviceById'
type MockAPIServer_ReadDeviceById_Call struct {
	*mock.Call
}

// ReadDeviceById is a helper method to define mock.On call
//   - ctx context.Context
//   - deviceID int64
func (_e *MockAPIServer_Expecter) ReadDeviceById(ctx interface{}, deviceID interface{}) *MockAPIServer_ReadDeviceById_Call {
	return &MockAPIServer_ReadDeviceById_Call{Call: _e.mock.On("ReadDeviceById", ctx, deviceID)}
}

func (_c *MockAPIServer_ReadDeviceById_Call) Run(run func(ctx context.Context, deviceID int64)) *MockAPIServer_ReadDeviceById_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(int64))
	})
	return _c
}

func (_c *MockAPIServer_ReadDeviceById_Call) Return(_a0 *pb.Device, _a1 error) *MockAPIServer_ReadDeviceById_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAPIServer_ReadDeviceById_Call) RunAndReturn(run func(context.Context, int64) (*pb.Device, error)) *MockAPIServer_ReadDeviceById_Call {
	_c.Call.Return(run)
	return _c
}

// ReadDeviceBySerialPlatform provides a mock function with given fields: ctx, serial, platform
func (_m *MockAPIServer) ReadDeviceBySerialPlatform(ctx context.Context, serial string, platform string) (*pb.Device, error) {
	ret := _m.Called(ctx, serial, platform)

	if len(ret) == 0 {
		panic("no return value specified for ReadDeviceBySerialPlatform")
	}

	var r0 *pb.Device
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (*pb.Device, error)); ok {
		return rf(ctx, serial, platform)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *pb.Device); ok {
		r0 = rf(ctx, serial, platform)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Device)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, serial, platform)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAPIServer_ReadDeviceBySerialPlatform_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReadDeviceBySerialPlatform'
type MockAPIServer_ReadDeviceBySerialPlatform_Call struct {
	*mock.Call
}

// ReadDeviceBySerialPlatform is a helper method to define mock.On call
//   - ctx context.Context
//   - serial string
//   - platform string
func (_e *MockAPIServer_Expecter) ReadDeviceBySerialPlatform(ctx interface{}, serial interface{}, platform interface{}) *MockAPIServer_ReadDeviceBySerialPlatform_Call {
	return &MockAPIServer_ReadDeviceBySerialPlatform_Call{Call: _e.mock.On("ReadDeviceBySerialPlatform", ctx, serial, platform)}
}

func (_c *MockAPIServer_ReadDeviceBySerialPlatform_Call) Run(run func(ctx context.Context, serial string, platform string)) *MockAPIServer_ReadDeviceBySerialPlatform_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *MockAPIServer_ReadDeviceBySerialPlatform_Call) Return(_a0 *pb.Device, _a1 error) *MockAPIServer_ReadDeviceBySerialPlatform_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAPIServer_ReadDeviceBySerialPlatform_Call) RunAndReturn(run func(context.Context, string, string) (*pb.Device, error)) *MockAPIServer_ReadDeviceBySerialPlatform_Call {
	_c.Call.Return(run)
	return _c
}

// ReadDevices provides a mock function with given fields: ctx
func (_m *MockAPIServer) ReadDevices(ctx context.Context) ([]*pb.Device, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for ReadDevices")
	}

	var r0 []*pb.Device
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]*pb.Device, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []*pb.Device); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pb.Device)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAPIServer_ReadDevices_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReadDevices'
type MockAPIServer_ReadDevices_Call struct {
	*mock.Call
}

// ReadDevices is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockAPIServer_Expecter) ReadDevices(ctx interface{}) *MockAPIServer_ReadDevices_Call {
	return &MockAPIServer_ReadDevices_Call{Call: _e.mock.On("ReadDevices", ctx)}
}

func (_c *MockAPIServer_ReadDevices_Call) Run(run func(ctx context.Context)) *MockAPIServer_ReadDevices_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockAPIServer_ReadDevices_Call) Return(_a0 []*pb.Device, _a1 error) *MockAPIServer_ReadDevices_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAPIServer_ReadDevices_Call) RunAndReturn(run func(context.Context) ([]*pb.Device, error)) *MockAPIServer_ReadDevices_Call {
	_c.Call.Return(run)
	return _c
}

// ReadGateway provides a mock function with given fields: ctx, name
func (_m *MockAPIServer) ReadGateway(ctx context.Context, name string) (*pb.Gateway, error) {
	ret := _m.Called(ctx, name)

	if len(ret) == 0 {
		panic("no return value specified for ReadGateway")
	}

	var r0 *pb.Gateway
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*pb.Gateway, error)); ok {
		return rf(ctx, name)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *pb.Gateway); ok {
		r0 = rf(ctx, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Gateway)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAPIServer_ReadGateway_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReadGateway'
type MockAPIServer_ReadGateway_Call struct {
	*mock.Call
}

// ReadGateway is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
func (_e *MockAPIServer_Expecter) ReadGateway(ctx interface{}, name interface{}) *MockAPIServer_ReadGateway_Call {
	return &MockAPIServer_ReadGateway_Call{Call: _e.mock.On("ReadGateway", ctx, name)}
}

func (_c *MockAPIServer_ReadGateway_Call) Run(run func(ctx context.Context, name string)) *MockAPIServer_ReadGateway_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockAPIServer_ReadGateway_Call) Return(_a0 *pb.Gateway, _a1 error) *MockAPIServer_ReadGateway_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAPIServer_ReadGateway_Call) RunAndReturn(run func(context.Context, string) (*pb.Gateway, error)) *MockAPIServer_ReadGateway_Call {
	_c.Call.Return(run)
	return _c
}

// ReadGateways provides a mock function with given fields: ctx
func (_m *MockAPIServer) ReadGateways(ctx context.Context) ([]*pb.Gateway, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for ReadGateways")
	}

	var r0 []*pb.Gateway
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]*pb.Gateway, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []*pb.Gateway); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pb.Gateway)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAPIServer_ReadGateways_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReadGateways'
type MockAPIServer_ReadGateways_Call struct {
	*mock.Call
}

// ReadGateways is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockAPIServer_Expecter) ReadGateways(ctx interface{}) *MockAPIServer_ReadGateways_Call {
	return &MockAPIServer_ReadGateways_Call{Call: _e.mock.On("ReadGateways", ctx)}
}

func (_c *MockAPIServer_ReadGateways_Call) Run(run func(ctx context.Context)) *MockAPIServer_ReadGateways_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockAPIServer_ReadGateways_Call) Return(_a0 []*pb.Gateway, _a1 error) *MockAPIServer_ReadGateways_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAPIServer_ReadGateways_Call) RunAndReturn(run func(context.Context) ([]*pb.Gateway, error)) *MockAPIServer_ReadGateways_Call {
	_c.Call.Return(run)
	return _c
}

// ReadMostRecentSessionInfo provides a mock function with given fields: ctx, deviceID
func (_m *MockAPIServer) ReadMostRecentSessionInfo(ctx context.Context, deviceID int64) (*pb.Session, error) {
	ret := _m.Called(ctx, deviceID)

	if len(ret) == 0 {
		panic("no return value specified for ReadMostRecentSessionInfo")
	}

	var r0 *pb.Session
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int64) (*pb.Session, error)); ok {
		return rf(ctx, deviceID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int64) *pb.Session); ok {
		r0 = rf(ctx, deviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Session)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, int64) error); ok {
		r1 = rf(ctx, deviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAPIServer_ReadMostRecentSessionInfo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReadMostRecentSessionInfo'
type MockAPIServer_ReadMostRecentSessionInfo_Call struct {
	*mock.Call
}

// ReadMostRecentSessionInfo is a helper method to define mock.On call
//   - ctx context.Context
//   - deviceID int64
func (_e *MockAPIServer_Expecter) ReadMostRecentSessionInfo(ctx interface{}, deviceID interface{}) *MockAPIServer_ReadMostRecentSessionInfo_Call {
	return &MockAPIServer_ReadMostRecentSessionInfo_Call{Call: _e.mock.On("ReadMostRecentSessionInfo", ctx, deviceID)}
}

func (_c *MockAPIServer_ReadMostRecentSessionInfo_Call) Run(run func(ctx context.Context, deviceID int64)) *MockAPIServer_ReadMostRecentSessionInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(int64))
	})
	return _c
}

func (_c *MockAPIServer_ReadMostRecentSessionInfo_Call) Return(_a0 *pb.Session, _a1 error) *MockAPIServer_ReadMostRecentSessionInfo_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAPIServer_ReadMostRecentSessionInfo_Call) RunAndReturn(run func(context.Context, int64) (*pb.Session, error)) *MockAPIServer_ReadMostRecentSessionInfo_Call {
	_c.Call.Return(run)
	return _c
}

// ReadSessionInfo provides a mock function with given fields: ctx, key
func (_m *MockAPIServer) ReadSessionInfo(ctx context.Context, key string) (*pb.Session, error) {
	ret := _m.Called(ctx, key)

	if len(ret) == 0 {
		panic("no return value specified for ReadSessionInfo")
	}

	var r0 *pb.Session
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*pb.Session, error)); ok {
		return rf(ctx, key)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *pb.Session); ok {
		r0 = rf(ctx, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Session)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAPIServer_ReadSessionInfo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReadSessionInfo'
type MockAPIServer_ReadSessionInfo_Call struct {
	*mock.Call
}

// ReadSessionInfo is a helper method to define mock.On call
//   - ctx context.Context
//   - key string
func (_e *MockAPIServer_Expecter) ReadSessionInfo(ctx interface{}, key interface{}) *MockAPIServer_ReadSessionInfo_Call {
	return &MockAPIServer_ReadSessionInfo_Call{Call: _e.mock.On("ReadSessionInfo", ctx, key)}
}

func (_c *MockAPIServer_ReadSessionInfo_Call) Run(run func(ctx context.Context, key string)) *MockAPIServer_ReadSessionInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockAPIServer_ReadSessionInfo_Call) Return(_a0 *pb.Session, _a1 error) *MockAPIServer_ReadSessionInfo_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAPIServer_ReadSessionInfo_Call) RunAndReturn(run func(context.Context, string) (*pb.Session, error)) *MockAPIServer_ReadSessionInfo_Call {
	_c.Call.Return(run)
	return _c
}

// ReadSessionInfos provides a mock function with given fields: ctx
func (_m *MockAPIServer) ReadSessionInfos(ctx context.Context) ([]*pb.Session, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for ReadSessionInfos")
	}

	var r0 []*pb.Session
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]*pb.Session, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []*pb.Session); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pb.Session)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAPIServer_ReadSessionInfos_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReadSessionInfos'
type MockAPIServer_ReadSessionInfos_Call struct {
	*mock.Call
}

// ReadSessionInfos is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockAPIServer_Expecter) ReadSessionInfos(ctx interface{}) *MockAPIServer_ReadSessionInfos_Call {
	return &MockAPIServer_ReadSessionInfos_Call{Call: _e.mock.On("ReadSessionInfos", ctx)}
}

func (_c *MockAPIServer_ReadSessionInfos_Call) Run(run func(ctx context.Context)) *MockAPIServer_ReadSessionInfos_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockAPIServer_ReadSessionInfos_Call) Return(_a0 []*pb.Session, _a1 error) *MockAPIServer_ReadSessionInfos_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAPIServer_ReadSessionInfos_Call) RunAndReturn(run func(context.Context) ([]*pb.Session, error)) *MockAPIServer_ReadSessionInfos_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveExpiredSessions provides a mock function with given fields: ctx
func (_m *MockAPIServer) RemoveExpiredSessions(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for RemoveExpiredSessions")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAPIServer_RemoveExpiredSessions_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveExpiredSessions'
type MockAPIServer_RemoveExpiredSessions_Call struct {
	*mock.Call
}

// RemoveExpiredSessions is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockAPIServer_Expecter) RemoveExpiredSessions(ctx interface{}) *MockAPIServer_RemoveExpiredSessions_Call {
	return &MockAPIServer_RemoveExpiredSessions_Call{Call: _e.mock.On("RemoveExpiredSessions", ctx)}
}

func (_c *MockAPIServer_RemoveExpiredSessions_Call) Run(run func(ctx context.Context)) *MockAPIServer_RemoveExpiredSessions_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockAPIServer_RemoveExpiredSessions_Call) Return(_a0 error) *MockAPIServer_RemoveExpiredSessions_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAPIServer_RemoveExpiredSessions_Call) RunAndReturn(run func(context.Context) error) *MockAPIServer_RemoveExpiredSessions_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateDevices provides a mock function with given fields: ctx, devices
func (_m *MockAPIServer) UpdateDevices(ctx context.Context, devices []*pb.Device) error {
	ret := _m.Called(ctx, devices)

	if len(ret) == 0 {
		panic("no return value specified for UpdateDevices")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, []*pb.Device) error); ok {
		r0 = rf(ctx, devices)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAPIServer_UpdateDevices_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateDevices'
type MockAPIServer_UpdateDevices_Call struct {
	*mock.Call
}

// UpdateDevices is a helper method to define mock.On call
//   - ctx context.Context
//   - devices []*pb.Device
func (_e *MockAPIServer_Expecter) UpdateDevices(ctx interface{}, devices interface{}) *MockAPIServer_UpdateDevices_Call {
	return &MockAPIServer_UpdateDevices_Call{Call: _e.mock.On("UpdateDevices", ctx, devices)}
}

func (_c *MockAPIServer_UpdateDevices_Call) Run(run func(ctx context.Context, devices []*pb.Device)) *MockAPIServer_UpdateDevices_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]*pb.Device))
	})
	return _c
}

func (_c *MockAPIServer_UpdateDevices_Call) Return(_a0 error) *MockAPIServer_UpdateDevices_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAPIServer_UpdateDevices_Call) RunAndReturn(run func(context.Context, []*pb.Device) error) *MockAPIServer_UpdateDevices_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateGateway provides a mock function with given fields: ctx, gateway
func (_m *MockAPIServer) UpdateGateway(ctx context.Context, gateway *pb.Gateway) error {
	ret := _m.Called(ctx, gateway)

	if len(ret) == 0 {
		panic("no return value specified for UpdateGateway")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *pb.Gateway) error); ok {
		r0 = rf(ctx, gateway)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAPIServer_UpdateGateway_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateGateway'
type MockAPIServer_UpdateGateway_Call struct {
	*mock.Call
}

// UpdateGateway is a helper method to define mock.On call
//   - ctx context.Context
//   - gateway *pb.Gateway
func (_e *MockAPIServer_Expecter) UpdateGateway(ctx interface{}, gateway interface{}) *MockAPIServer_UpdateGateway_Call {
	return &MockAPIServer_UpdateGateway_Call{Call: _e.mock.On("UpdateGateway", ctx, gateway)}
}

func (_c *MockAPIServer_UpdateGateway_Call) Run(run func(ctx context.Context, gateway *pb.Gateway)) *MockAPIServer_UpdateGateway_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*pb.Gateway))
	})
	return _c
}

func (_c *MockAPIServer_UpdateGateway_Call) Return(_a0 error) *MockAPIServer_UpdateGateway_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAPIServer_UpdateGateway_Call) RunAndReturn(run func(context.Context, *pb.Gateway) error) *MockAPIServer_UpdateGateway_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateGatewayDynamicFields provides a mock function with given fields: ctx, gateway
func (_m *MockAPIServer) UpdateGatewayDynamicFields(ctx context.Context, gateway *pb.Gateway) error {
	ret := _m.Called(ctx, gateway)

	if len(ret) == 0 {
		panic("no return value specified for UpdateGatewayDynamicFields")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *pb.Gateway) error); ok {
		r0 = rf(ctx, gateway)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAPIServer_UpdateGatewayDynamicFields_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateGatewayDynamicFields'
type MockAPIServer_UpdateGatewayDynamicFields_Call struct {
	*mock.Call
}

// UpdateGatewayDynamicFields is a helper method to define mock.On call
//   - ctx context.Context
//   - gateway *pb.Gateway
func (_e *MockAPIServer_Expecter) UpdateGatewayDynamicFields(ctx interface{}, gateway interface{}) *MockAPIServer_UpdateGatewayDynamicFields_Call {
	return &MockAPIServer_UpdateGatewayDynamicFields_Call{Call: _e.mock.On("UpdateGatewayDynamicFields", ctx, gateway)}
}

func (_c *MockAPIServer_UpdateGatewayDynamicFields_Call) Run(run func(ctx context.Context, gateway *pb.Gateway)) *MockAPIServer_UpdateGatewayDynamicFields_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*pb.Gateway))
	})
	return _c
}

func (_c *MockAPIServer_UpdateGatewayDynamicFields_Call) Return(_a0 error) *MockAPIServer_UpdateGatewayDynamicFields_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAPIServer_UpdateGatewayDynamicFields_Call) RunAndReturn(run func(context.Context, *pb.Gateway) error) *MockAPIServer_UpdateGatewayDynamicFields_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateSingleDevice provides a mock function with given fields: ctx, externalID, serial, platform, lastSeen, issues
func (_m *MockAPIServer) UpdateSingleDevice(ctx context.Context, externalID string, serial string, platform string, lastSeen *time.Time, issues []*pb.DeviceIssue) (int64, error) {
	ret := _m.Called(ctx, externalID, serial, platform, lastSeen, issues)

	if len(ret) == 0 {
		panic("no return value specified for UpdateSingleDevice")
	}

	var r0 int64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, *time.Time, []*pb.DeviceIssue) (int64, error)); ok {
		return rf(ctx, externalID, serial, platform, lastSeen, issues)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, *time.Time, []*pb.DeviceIssue) int64); ok {
		r0 = rf(ctx, externalID, serial, platform, lastSeen, issues)
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, *time.Time, []*pb.DeviceIssue) error); ok {
		r1 = rf(ctx, externalID, serial, platform, lastSeen, issues)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAPIServer_UpdateSingleDevice_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateSingleDevice'
type MockAPIServer_UpdateSingleDevice_Call struct {
	*mock.Call
}

// UpdateSingleDevice is a helper method to define mock.On call
//   - ctx context.Context
//   - externalID string
//   - serial string
//   - platform string
//   - lastSeen *time.Time
//   - issues []*pb.DeviceIssue
func (_e *MockAPIServer_Expecter) UpdateSingleDevice(ctx interface{}, externalID interface{}, serial interface{}, platform interface{}, lastSeen interface{}, issues interface{}) *MockAPIServer_UpdateSingleDevice_Call {
	return &MockAPIServer_UpdateSingleDevice_Call{Call: _e.mock.On("UpdateSingleDevice", ctx, externalID, serial, platform, lastSeen, issues)}
}

func (_c *MockAPIServer_UpdateSingleDevice_Call) Run(run func(ctx context.Context, externalID string, serial string, platform string, lastSeen *time.Time, issues []*pb.DeviceIssue)) *MockAPIServer_UpdateSingleDevice_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(string), args[4].(*time.Time), args[5].([]*pb.DeviceIssue))
	})
	return _c
}

func (_c *MockAPIServer_UpdateSingleDevice_Call) Return(_a0 int64, _a1 error) *MockAPIServer_UpdateSingleDevice_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAPIServer_UpdateSingleDevice_Call) RunAndReturn(run func(context.Context, string, string, string, *time.Time, []*pb.DeviceIssue) (int64, error)) *MockAPIServer_UpdateSingleDevice_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockAPIServer creates a new instance of MockAPIServer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockAPIServer(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockAPIServer {
	mock := &MockAPIServer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
