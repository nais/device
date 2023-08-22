// Code generated by mockery. DO NOT EDIT.

package pb

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	metadata "google.golang.org/grpc/metadata"
)

// MockAPIServer_GetGatewayConfigurationClient is an autogenerated mock type for the APIServer_GetGatewayConfigurationClient type
type MockAPIServer_GetGatewayConfigurationClient struct {
	mock.Mock
}

// CloseSend provides a mock function with given fields:
func (_m *MockAPIServer_GetGatewayConfigurationClient) CloseSend() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Context provides a mock function with given fields:
func (_m *MockAPIServer_GetGatewayConfigurationClient) Context() context.Context {
	ret := _m.Called()

	var r0 context.Context
	if rf, ok := ret.Get(0).(func() context.Context); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(context.Context)
		}
	}

	return r0
}

// Header provides a mock function with given fields:
func (_m *MockAPIServer_GetGatewayConfigurationClient) Header() (metadata.MD, error) {
	ret := _m.Called()

	var r0 metadata.MD
	var r1 error
	if rf, ok := ret.Get(0).(func() (metadata.MD, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() metadata.MD); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metadata.MD)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Recv provides a mock function with given fields:
func (_m *MockAPIServer_GetGatewayConfigurationClient) Recv() (*GetGatewayConfigurationResponse, error) {
	ret := _m.Called()

	var r0 *GetGatewayConfigurationResponse
	var r1 error
	if rf, ok := ret.Get(0).(func() (*GetGatewayConfigurationResponse, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() *GetGatewayConfigurationResponse); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*GetGatewayConfigurationResponse)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RecvMsg provides a mock function with given fields: m
func (_m *MockAPIServer_GetGatewayConfigurationClient) RecvMsg(m interface{}) error {
	ret := _m.Called(m)

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(m)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SendMsg provides a mock function with given fields: m
func (_m *MockAPIServer_GetGatewayConfigurationClient) SendMsg(m interface{}) error {
	ret := _m.Called(m)

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(m)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Trailer provides a mock function with given fields:
func (_m *MockAPIServer_GetGatewayConfigurationClient) Trailer() metadata.MD {
	ret := _m.Called()

	var r0 metadata.MD
	if rf, ok := ret.Get(0).(func() metadata.MD); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metadata.MD)
		}
	}

	return r0
}

// NewMockAPIServer_GetGatewayConfigurationClient creates a new instance of MockAPIServer_GetGatewayConfigurationClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockAPIServer_GetGatewayConfigurationClient(t interface {
	mock.TestingT
	Cleanup(func())
},
) *MockAPIServer_GetGatewayConfigurationClient {
	mock := &MockAPIServer_GetGatewayConfigurationClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
