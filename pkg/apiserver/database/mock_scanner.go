// Code generated by mockery v2.30.1. DO NOT EDIT.

package database

import mock "github.com/stretchr/testify/mock"

// MockScanner is an autogenerated mock type for the Scanner type
type MockScanner struct {
	mock.Mock
}

// Scan provides a mock function with given fields: _a0
func (_m *MockScanner) Scan(_a0 ...interface{}) error {
	var _ca []interface{}
	_ca = append(_ca, _a0...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(...interface{}) error); ok {
		r0 = rf(_a0...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockScanner creates a new instance of MockScanner. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockScanner(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockScanner {
	mock := &MockScanner{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
