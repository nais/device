// Code generated by mockery v2.32.2. DO NOT EDIT.

package pb

import mock "github.com/stretchr/testify/mock"

// MockUnsafeDeviceHelperServer is an autogenerated mock type for the UnsafeDeviceHelperServer type
type MockUnsafeDeviceHelperServer struct {
	mock.Mock
}

// mustEmbedUnimplementedDeviceHelperServer provides a mock function with given fields:
func (_m *MockUnsafeDeviceHelperServer) mustEmbedUnimplementedDeviceHelperServer() {
	_m.Called()
}

// NewMockUnsafeDeviceHelperServer creates a new instance of MockUnsafeDeviceHelperServer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockUnsafeDeviceHelperServer(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockUnsafeDeviceHelperServer {
	mock := &MockUnsafeDeviceHelperServer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
