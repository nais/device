//go:build !tenant
// +build !tenant

package runtimeconfig

import "github.com/nais/device/internal/pb"

var defaultTenants = []*pb.Tenant{
	{
		Name:         "NAV",
		AuthProvider: pb.AuthProvider_Azure,
		Active:       true,
	},
}
