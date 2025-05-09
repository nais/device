package helper

import (
	"context"

	"github.com/nais/device/internal/otel"
	"github.com/nais/device/pkg/pb"
	ototel "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type TracedConfigurator struct {
	Wrapped OSConfigurator

	counter       metric.Int64Counter
	routesCounter metric.Int64Histogram
}

var _ OSConfigurator = &TracedConfigurator{}

func NewTracedConfigurator(w OSConfigurator) *TracedConfigurator {
	meter := ototel.Meter("configurator")
	counter, err := meter.Int64Counter("configurator_calls",
		metric.WithDescription("The number of calls to the configurator"),
		metric.WithUnit("{call}"))
	if err != nil {
		panic(err)
	}

	routesCounter, err := meter.Int64Histogram("configurator_routes",
		metric.WithDescription("The number of routes created by the configurator"),
		metric.WithExplicitBucketBoundaries(20, 40, 60, 80, 100, 120, 140, 160, 180, 200),
	)
	if err != nil {
		panic(err)
	}

	return &TracedConfigurator{
		Wrapped:       w,
		counter:       counter,
		routesCounter: routesCounter,
	}
}

func (tc *TracedConfigurator) SetupInterface(ctx context.Context, cfg *pb.Configuration) error {
	ctx, span := otel.Start(ctx, "SetupInterface")
	defer span.End()
	err := tc.Wrapped.SetupInterface(ctx, cfg)
	span.RecordError(err)

	tc.counter.Add(ctx, 1, metric.WithAttributes(attribute.String("method", "SetupInterface"), attribute.Bool("error", err != nil)))
	return err
}

func (tc *TracedConfigurator) TeardownInterface(ctx context.Context) error {
	ctx, span := otel.Start(ctx, "TeardownInterface")
	defer span.End()
	err := tc.Wrapped.TeardownInterface(ctx)
	span.RecordError(err)

	tc.counter.Add(ctx, 1, metric.WithAttributes(attribute.String("method", "TeardownInterface"), attribute.Bool("error", err != nil)))
	return err
}

func (tc *TracedConfigurator) SyncConf(ctx context.Context, cfg *pb.Configuration) error {
	ctx, span := otel.Start(ctx, "SyncConf")
	defer span.End()
	err := tc.Wrapped.SyncConf(ctx, cfg)
	span.RecordError(err)

	tc.counter.Add(ctx, 1, metric.WithAttributes(attribute.String("method", "SyncConf"), attribute.Bool("error", err != nil)))
	return err
}

func (tc *TracedConfigurator) SetupRoutes(ctx context.Context, gateways []*pb.Gateway) (int, error) {
	ctx, span := otel.Start(ctx, "SetupRoutes")
	defer span.End()
	i, err := tc.Wrapped.SetupRoutes(ctx, gateways)
	span.RecordError(err)
	span.SetAttributes(attribute.Int("routes", i))

	tc.counter.Add(ctx, 1, metric.WithAttributes(attribute.String("method", "SetupRoutes"), attribute.Bool("error", err != nil)))
	tc.routesCounter.Record(ctx, int64(i))
	return i, err
}

func (tc *TracedConfigurator) Prerequisites() error {
	err := tc.Wrapped.Prerequisites()
	tc.counter.Add(context.Background(), 1, metric.WithAttributes(attribute.String("method", "Prerequisites"), attribute.Bool("error", err != nil)))
	return err
}
