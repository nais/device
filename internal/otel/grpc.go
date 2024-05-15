package otel

import (
	"context"
	"slices"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc/stats"
)

type grpcClientHandler struct {
	stats.Handler

	ignore []string
}

// NewGRPCClientHandler creates a new stats.Handler that ignores tracing/metrics for the provided methods.
func NewGRPCClientHandler(ignore ...string) stats.Handler {
	return &grpcClientHandler{
		Handler: otelgrpc.NewClientHandler(),
		ignore:  ignore,
	}
}

func (h *grpcClientHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	if slices.Contains(h.ignore, info.FullMethodName) {
		return ctx
	}

	return h.Handler.TagRPC(ctx, info)
}
