package helper

import (
	"context"

	"github.com/nais/device/internal/otel"
	"github.com/nais/device/internal/pb"
)

type TracedConfigurator struct {
	Wrapped OSConfigurator
}

var _ OSConfigurator = &TracedConfigurator{}

func (tc *TracedConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	ctx, span := otel.Start(ctx, "SetupInterface")
	defer span.End()
	return tc.Wrapped.SetupInterface(ctx, cfg)
}

func (tc *TracedConfigurator) TeardownInterface(ctx context.Context) error {
	ctx, span := otel.Start(ctx, "TeardownInterface")
	defer span.End()
	return tc.Wrapped.TeardownInterface(ctx)
}

func (tc *TracedConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	ctx, span := otel.Start(ctx, "SyncConf")
	defer span.End()
	return tc.Wrapped.SyncConf(ctx, cfg)
}

func (tc *TracedConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) error {
	ctx, span := otel.Start(ctx, "SetupRoutes")
	defer span.End()
	return tc.Wrapped.SetupRoutes(ctx, gateways)
}

func (tc *TracedConfigurator) Prerequisites() error {
	return tc.Wrapped.Prerequisites()
}
