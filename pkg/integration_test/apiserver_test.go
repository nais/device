package integrationtest_test

import (
	"context"
	"testing"

	"github.com/nais/device/pkg/apiserver/api"
	"github.com/nais/device/pkg/apiserver/auth"
	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/jita"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func NewAPIServer(t *testing.T, ctx context.Context, log *logrus.Entry, db database.APIServer) *grpc.Server {
	sessions := auth.NewSessionStore(db)
	deviceAuth := auth.NewMockAuthenticator(sessions)
	gatewayAuth := auth.NewMockAPIKeyAuthenticator()

	j := jita.New(log, "user", "pass", "url")

	impl := api.NewGRPCServer(ctx, log, db, deviceAuth, nil, gatewayAuth, nil, j, sessions)
	server := grpc.NewServer()
	pb.RegisterAPIServerServer(server, impl)

	return server
}
