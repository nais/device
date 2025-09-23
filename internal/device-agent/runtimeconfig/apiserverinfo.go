package runtimeconfig

import "github.com/nais/device/pkg/pb"

type ApiServerInfo struct {
	Client     pb.APIServerClient
	SessionKey string
}
