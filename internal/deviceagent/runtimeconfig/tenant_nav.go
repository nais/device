//go:build !tenant

package runtimeconfig

import "github.com/nais/device/pkg/pb"

var defaultTenants = []*pb.Tenant{
	{
		Name:         "NAV",
		AuthProvider: pb.AuthProvider_Azure,
		Active:       true,
	},
}
