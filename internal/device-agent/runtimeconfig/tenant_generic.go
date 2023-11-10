//go:build tenant
// +build tenant

package runtimeconfig

import "github.com/nais/device/internal/pb"

var defaultTenants = []*pb.Tenant{
	{
		Name:         "tenant",
		AuthProvider: pb.AuthProvider_Google,
	},
}
