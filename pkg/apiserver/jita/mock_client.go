// Code generated by mockery v2.32.2. DO NOT EDIT.

package jita

import mock "github.com/stretchr/testify/mock"

// MockClient is an autogenerated mock type for the Client type
type MockClient struct {
	mock.Mock
}

// GetPrivilegedUsersForGateway provides a mock function with given fields: gateway
func (_m *MockClient) GetPrivilegedUsersForGateway(gateway string) []PrivilegedUser {
	ret := _m.Called(gateway)

	var r0 []PrivilegedUser
	if rf, ok := ret.Get(0).(func(string) []PrivilegedUser); ok {
		r0 = rf(gateway)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]PrivilegedUser)
		}
	}

	return r0
}

// UpdatePrivilegedUsers provides a mock function with given fields:
func (_m *MockClient) UpdatePrivilegedUsers() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockClient creates a new instance of MockClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockClient {
	mock := &MockClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
