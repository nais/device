// Code generated by mockery v2.30.1. DO NOT EDIT.

package middleware

import (
	http "net/http"

	mock "github.com/stretchr/testify/mock"
)

// mockMiddleware is an autogenerated mock type for the middleware type
type mockMiddleware struct {
	mock.Mock
}

// Execute provides a mock function with given fields: _a0
func (_m *mockMiddleware) Execute(_a0 http.Handler) http.Handler {
	ret := _m.Called(_a0)

	var r0 http.Handler
	if rf, ok := ret.Get(0).(func(http.Handler) http.Handler); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(http.Handler)
		}
	}

	return r0
}

// newMockMiddleware creates a new instance of mockMiddleware. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockMiddleware(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockMiddleware {
	mock := &mockMiddleware{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
