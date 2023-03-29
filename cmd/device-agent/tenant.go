//go:build tenant
// +build tenant

package main

import "github.com/nais/device/pkg/pb"

var defaultTenant = []*pb.Tenant{
	{
		Name:         "tenant",
		AuthProvider: pb.AuthProvider_Google,
	},
}
