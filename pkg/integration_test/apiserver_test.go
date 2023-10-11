package integrationtest_test

import (
	"context"
	"testing"

	"github.com/nais/device/pkg/apiserver/api"
	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/jita"
	"github.com/nais/device/pkg/pb"
	"google.golang.org/grpc"
)

func NewAPIServer(t *testing.T, ctx context.Context, db database.APIServer) *grpc.Server {
	sessions := auth.NewMockSessionStore(t)
	deviceAuth := auth.NewMockAuthenticator(sessions)
	gatewayAuth := auth.NewMockAPIKeyAuthenticator()

	j := jita.New("user", "pass", "url")

	impl := api.NewGRPCServer(ctx, db, deviceAuth, nil, gatewayAuth, nil, j, sessions)
	server := grpc.NewServer()
	pb.RegisterAPIServerServer(server, impl)

	return server
}
