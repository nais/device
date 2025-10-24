package pb

import (
	"context"

	"google.golang.org/grpc/stats"
)

// ConnectionNotifier implements gRPC stats.Handler and provides the two channels
// Connect() and Disconnect() to notify when gRPC connections are set up and torn down.
//
// Add to gRPC server with:
//
//	notifier := NewConnectionNotifier()
//	grpc.NewServer(grpc.StatsHandler(notifier))
//
// Listen for events with:
//
//	<- notifier.Connect()
//	<- notifier.Disconnect()
type ConnectionNotifier struct {
	connect    chan any
	disconnect chan any
}

func NewConnectionNotifier() *ConnectionNotifier {
	return &ConnectionNotifier{
		connect:    make(chan any, 16),
		disconnect: make(chan any, 16),
	}
}

func (h *ConnectionNotifier) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

func (h *ConnectionNotifier) HandleRPC(ctx context.Context, s stats.RPCStats) {}

func (h *ConnectionNotifier) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return context.TODO()
}

func (h *ConnectionNotifier) HandleConn(ctx context.Context, s stats.ConnStats) {
	switch s.(type) {
	case *stats.ConnBegin:
		h.connect <- new(any)
	case *stats.ConnEnd:
		h.disconnect <- new(any)
	}
}

func (h *ConnectionNotifier) Connect() <-chan any {
	return h.connect
}

func (h *ConnectionNotifier) Disconnect() <-chan any {
	return h.disconnect
}
