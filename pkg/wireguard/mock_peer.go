// Code generated by mockery v2.30.1. DO NOT EDIT.

package wireguard

import mock "github.com/stretchr/testify/mock"

// MockPeer is an autogenerated mock type for the Peer type
type MockPeer struct {
	mock.Mock
}

// GetAllowedIPs provides a mock function with given fields:
func (_m *MockPeer) GetAllowedIPs() []string {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// GetEndpoint provides a mock function with given fields:
func (_m *MockPeer) GetEndpoint() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetName provides a mock function with given fields:
func (_m *MockPeer) GetName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetPublicKey provides a mock function with given fields:
func (_m *MockPeer) GetPublicKey() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NewMockPeer creates a new instance of MockPeer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockPeer(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockPeer {
	mock := &MockPeer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
