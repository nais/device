// Code generated by mockery. DO NOT EDIT.

package auth

import (
	context "context"

	pb "github.com/nais/device/internal/pb"
	mock "github.com/stretchr/testify/mock"
)

// MockSessionStore is an autogenerated mock type for the SessionStore type
type MockSessionStore struct {
	mock.Mock
}

type MockSessionStore_Expecter struct {
	mock *mock.Mock
}

func (_m *MockSessionStore) EXPECT() *MockSessionStore_Expecter {
	return &MockSessionStore_Expecter{mock: &_m.Mock}
}

// All provides a mock function with given fields:
func (_m *MockSessionStore) All() []*pb.Session {
	ret := _m.Called()

	var r0 []*pb.Session
	if rf, ok := ret.Get(0).(func() []*pb.Session); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pb.Session)
		}
	}

	return r0
}

// MockSessionStore_All_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'All'
type MockSessionStore_All_Call struct {
	*mock.Call
}

// All is a helper method to define mock.On call
func (_e *MockSessionStore_Expecter) All() *MockSessionStore_All_Call {
	return &MockSessionStore_All_Call{Call: _e.mock.On("All")}
}

func (_c *MockSessionStore_All_Call) Run(run func()) *MockSessionStore_All_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockSessionStore_All_Call) Return(_a0 []*pb.Session) *MockSessionStore_All_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockSessionStore_All_Call) RunAndReturn(run func() []*pb.Session) *MockSessionStore_All_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: _a0, _a1
func (_m *MockSessionStore) Get(_a0 context.Context, _a1 string) (*pb.Session, error) {
	ret := _m.Called(_a0, _a1)

	var r0 *pb.Session
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*pb.Session, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *pb.Session); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.Session)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockSessionStore_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type MockSessionStore_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
func (_e *MockSessionStore_Expecter) Get(_a0 interface{}, _a1 interface{}) *MockSessionStore_Get_Call {
	return &MockSessionStore_Get_Call{Call: _e.mock.On("Get", _a0, _a1)}
}

func (_c *MockSessionStore_Get_Call) Run(run func(_a0 context.Context, _a1 string)) *MockSessionStore_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockSessionStore_Get_Call) Return(_a0 *pb.Session, _a1 error) *MockSessionStore_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockSessionStore_Get_Call) RunAndReturn(run func(context.Context, string) (*pb.Session, error)) *MockSessionStore_Get_Call {
	_c.Call.Return(run)
	return _c
}

// Set provides a mock function with given fields: _a0, _a1
func (_m *MockSessionStore) Set(_a0 context.Context, _a1 *pb.Session) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *pb.Session) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockSessionStore_Set_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Set'
type MockSessionStore_Set_Call struct {
	*mock.Call
}

// Set is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 *pb.Session
func (_e *MockSessionStore_Expecter) Set(_a0 interface{}, _a1 interface{}) *MockSessionStore_Set_Call {
	return &MockSessionStore_Set_Call{Call: _e.mock.On("Set", _a0, _a1)}
}

func (_c *MockSessionStore_Set_Call) Run(run func(_a0 context.Context, _a1 *pb.Session)) *MockSessionStore_Set_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*pb.Session))
	})
	return _c
}

func (_c *MockSessionStore_Set_Call) Return(_a0 error) *MockSessionStore_Set_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockSessionStore_Set_Call) RunAndReturn(run func(context.Context, *pb.Session) error) *MockSessionStore_Set_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockSessionStore creates a new instance of MockSessionStore. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockSessionStore(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockSessionStore {
	mock := &MockSessionStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}