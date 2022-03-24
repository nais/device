// Code generated by mockery v2.10.0. DO NOT EDIT.

package wireguard

import mock "github.com/stretchr/testify/mock"

// MockIPTables is an autogenerated mock type for the IPTables type
type MockIPTables struct {
	mock.Mock
}

// AppendUnique provides a mock function with given fields: table, chain, rulespec
func (_m *MockIPTables) AppendUnique(table string, chain string, rulespec ...string) error {
	_va := make([]interface{}, len(rulespec))
	for _i := range rulespec {
		_va[_i] = rulespec[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, table, chain)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, ...string) error); ok {
		r0 = rf(table, chain, rulespec...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ChangePolicy provides a mock function with given fields: table, chain, target
func (_m *MockIPTables) ChangePolicy(table string, chain string, target string) error {
	ret := _m.Called(table, chain, target)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(table, chain, target)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewChain provides a mock function with given fields: table, chain
func (_m *MockIPTables) NewChain(table string, chain string) error {
	ret := _m.Called(table, chain)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(table, chain)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
