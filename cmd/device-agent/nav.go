//go:build !tenant
// +build !tenant

package main

import "github.com/nais/device/pkg/pb"

var defaultTenant = []*pb.Tenant{
	{
		Name:         "NAV",
		AuthProvider: pb.AuthProvider_Azure,
	},
}
