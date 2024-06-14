package integrationtest_test

import (
	"context"
	"testing"

	"github.com/nais/device/internal/apiserver/api"
	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/jita"
	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func NewAPIServer(t *testing.T, ctx context.Context, log *logrus.Entry, db database.Database, kolideClient kolide.Client) *grpc.Server {
	sessions := auth.NewSessionStore(db)
	deviceAuth := auth.NewMockAuthenticator(sessions)
	gatewayAuth := auth.NewMockAPIKeyAuthenticator()

	j := jita.New(log, "user", "pass", "url")

	impl := api.NewGRPCServer(ctx, log, db, deviceAuth, nil, gatewayAuth, nil, j, sessions, kolideClient)
	server := grpc.NewServer()
	pb.RegisterAPIServerServer(server, impl)

	return server
}
