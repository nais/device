// Code generated by mockery. DO NOT EDIT.

package wireguard

import mock "github.com/stretchr/testify/mock"

// MockNetworkConfigurer is an autogenerated mock type for the NetworkConfigurer type
type MockNetworkConfigurer struct {
	mock.Mock
}

type MockNetworkConfigurer_Expecter struct {
	mock *mock.Mock
}

func (_m *MockNetworkConfigurer) EXPECT() *MockNetworkConfigurer_Expecter {
	return &MockNetworkConfigurer_Expecter{mock: &_m.Mock}
}

// ApplyWireGuardConfig provides a mock function with given fields: peers
func (_m *MockNetworkConfigurer) ApplyWireGuardConfig(peers []Peer) error {
	ret := _m.Called(peers)

	if len(ret) == 0 {
		panic("no return value specified for ApplyWireGuardConfig")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]Peer) error); ok {
		r0 = rf(peers)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockNetworkConfigurer_ApplyWireGuardConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ApplyWireGuardConfig'
type MockNetworkConfigurer_ApplyWireGuardConfig_Call struct {
	*mock.Call
}

// ApplyWireGuardConfig is a helper method to define mock.On call
//   - peers []Peer
func (_e *MockNetworkConfigurer_Expecter) ApplyWireGuardConfig(peers interface{}) *MockNetworkConfigurer_ApplyWireGuardConfig_Call {
	return &MockNetworkConfigurer_ApplyWireGuardConfig_Call{Call: _e.mock.On("ApplyWireGuardConfig", peers)}
}

func (_c *MockNetworkConfigurer_ApplyWireGuardConfig_Call) Run(run func(peers []Peer)) *MockNetworkConfigurer_ApplyWireGuardConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]Peer))
	})
	return _c
}

func (_c *MockNetworkConfigurer_ApplyWireGuardConfig_Call) Return(_a0 error) *MockNetworkConfigurer_ApplyWireGuardConfig_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockNetworkConfigurer_ApplyWireGuardConfig_Call) RunAndReturn(run func([]Peer) error) *MockNetworkConfigurer_ApplyWireGuardConfig_Call {
	_c.Call.Return(run)
	return _c
}

// ForwardRoutesV4 provides a mock function with given fields: routes
func (_m *MockNetworkConfigurer) ForwardRoutesV4(routes []string) error {
	ret := _m.Called(routes)

	if len(ret) == 0 {
		panic("no return value specified for ForwardRoutesV4")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]string) error); ok {
		r0 = rf(routes)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockNetworkConfigurer_ForwardRoutesV4_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ForwardRoutesV4'
type MockNetworkConfigurer_ForwardRoutesV4_Call struct {
	*mock.Call
}

// ForwardRoutesV4 is a helper method to define mock.On call
//   - routes []string
func (_e *MockNetworkConfigurer_Expecter) ForwardRoutesV4(routes interface{}) *MockNetworkConfigurer_ForwardRoutesV4_Call {
	return &MockNetworkConfigurer_ForwardRoutesV4_Call{Call: _e.mock.On("ForwardRoutesV4", routes)}
}

func (_c *MockNetworkConfigurer_ForwardRoutesV4_Call) Run(run func(routes []string)) *MockNetworkConfigurer_ForwardRoutesV4_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]string))
	})
	return _c
}

func (_c *MockNetworkConfigurer_ForwardRoutesV4_Call) Return(_a0 error) *MockNetworkConfigurer_ForwardRoutesV4_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockNetworkConfigurer_ForwardRoutesV4_Call) RunAndReturn(run func([]string) error) *MockNetworkConfigurer_ForwardRoutesV4_Call {
	_c.Call.Return(run)
	return _c
}

// ForwardRoutesV6 provides a mock function with given fields: routes
func (_m *MockNetworkConfigurer) ForwardRoutesV6(routes []string) error {
	ret := _m.Called(routes)

	if len(ret) == 0 {
		panic("no return value specified for ForwardRoutesV6")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]string) error); ok {
		r0 = rf(routes)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockNetworkConfigurer_ForwardRoutesV6_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ForwardRoutesV6'
type MockNetworkConfigurer_ForwardRoutesV6_Call struct {
	*mock.Call
}

// ForwardRoutesV6 is a helper method to define mock.On call
//   - routes []string
func (_e *MockNetworkConfigurer_Expecter) ForwardRoutesV6(routes interface{}) *MockNetworkConfigurer_ForwardRoutesV6_Call {
	return &MockNetworkConfigurer_ForwardRoutesV6_Call{Call: _e.mock.On("ForwardRoutesV6", routes)}
}

func (_c *MockNetworkConfigurer_ForwardRoutesV6_Call) Run(run func(routes []string)) *MockNetworkConfigurer_ForwardRoutesV6_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]string))
	})
	return _c
}

func (_c *MockNetworkConfigurer_ForwardRoutesV6_Call) Return(_a0 error) *MockNetworkConfigurer_ForwardRoutesV6_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockNetworkConfigurer_ForwardRoutesV6_Call) RunAndReturn(run func([]string) error) *MockNetworkConfigurer_ForwardRoutesV6_Call {
	_c.Call.Return(run)
	return _c
}

// SetupIPTables provides a mock function with given fields:
func (_m *MockNetworkConfigurer) SetupIPTables() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for SetupIPTables")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockNetworkConfigurer_SetupIPTables_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetupIPTables'
type MockNetworkConfigurer_SetupIPTables_Call struct {
	*mock.Call
}

// SetupIPTables is a helper method to define mock.On call
func (_e *MockNetworkConfigurer_Expecter) SetupIPTables() *MockNetworkConfigurer_SetupIPTables_Call {
	return &MockNetworkConfigurer_SetupIPTables_Call{Call: _e.mock.On("SetupIPTables")}
}

func (_c *MockNetworkConfigurer_SetupIPTables_Call) Run(run func()) *MockNetworkConfigurer_SetupIPTables_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockNetworkConfigurer_SetupIPTables_Call) Return(_a0 error) *MockNetworkConfigurer_SetupIPTables_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockNetworkConfigurer_SetupIPTables_Call) RunAndReturn(run func() error) *MockNetworkConfigurer_SetupIPTables_Call {
	_c.Call.Return(run)
	return _c
}

// SetupInterface provides a mock function with given fields:
func (_m *MockNetworkConfigurer) SetupInterface() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for SetupInterface")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockNetworkConfigurer_SetupInterface_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetupInterface'
type MockNetworkConfigurer_SetupInterface_Call struct {
	*mock.Call
}

// SetupInterface is a helper method to define mock.On call
func (_e *MockNetworkConfigurer_Expecter) SetupInterface() *MockNetworkConfigurer_SetupInterface_Call {
	return &MockNetworkConfigurer_SetupInterface_Call{Call: _e.mock.On("SetupInterface")}
}

func (_c *MockNetworkConfigurer_SetupInterface_Call) Run(run func()) *MockNetworkConfigurer_SetupInterface_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockNetworkConfigurer_SetupInterface_Call) Return(_a0 error) *MockNetworkConfigurer_SetupInterface_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockNetworkConfigurer_SetupInterface_Call) RunAndReturn(run func() error) *MockNetworkConfigurer_SetupInterface_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockNetworkConfigurer creates a new instance of MockNetworkConfigurer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockNetworkConfigurer(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockNetworkConfigurer {
	mock := &MockNetworkConfigurer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
