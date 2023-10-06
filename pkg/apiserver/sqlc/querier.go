// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0

package sqlc

import (
	"context"
)

type Querier interface {
	AddDevice(ctx context.Context, arg AddDeviceParams) error
	AddGateway(ctx context.Context, arg AddGatewayParams) error
	AddGatewayAccessGroupID(ctx context.Context, arg AddGatewayAccessGroupIDParams) error
	AddGatewayRoute(ctx context.Context, arg AddGatewayRouteParams) error
	AddSession(ctx context.Context, arg AddSessionParams) error
	AddSessionAccessGroupID(ctx context.Context, arg AddSessionAccessGroupIDParams) error
	DeleteGatewayAccessGroupIDs(ctx context.Context, gatewayName string) error
	DeleteGatewayRoutes(ctx context.Context, gatewayName string) error
	GetDeviceByID(ctx context.Context, id int64) (*Device, error)
	GetDeviceByPublicKey(ctx context.Context, publicKey string) (*Device, error)
	GetDeviceBySerialAndPlatform(ctx context.Context, arg GetDeviceBySerialAndPlatformParams) (*Device, error)
	GetDevices(ctx context.Context) ([]*Device, error)
	GetGatewayAccessGroupIDs(ctx context.Context, gatewayName string) ([]string, error)
	GetGatewayByName(ctx context.Context, name string) (*Gateway, error)
	GetGatewayRoutes(ctx context.Context, gatewayName string) ([]*GetGatewayRoutesRow, error)
	GetGateways(ctx context.Context) ([]*Gateway, error)
	GetLastUsedIPV6(ctx context.Context) (string, error)
	GetMostRecentDeviceSession(ctx context.Context, sessionDeviceID int64) (*GetMostRecentDeviceSessionRow, error)
	GetSessionByKey(ctx context.Context, sessionKey string) (*GetSessionByKeyRow, error)
	GetSessionGroupIDs(ctx context.Context, sessionKey string) ([]string, error)
	GetSessions(ctx context.Context) ([]*GetSessionsRow, error)
	RemoveExpiredSessions(ctx context.Context) error
	UpdateDevice(ctx context.Context, arg UpdateDeviceParams) error
	UpdateGateway(ctx context.Context, arg UpdateGatewayParams) error
	UpdateGatewayDynamicFields(ctx context.Context, arg UpdateGatewayDynamicFieldsParams) error
}

var _ Querier = (*Queries)(nil)
