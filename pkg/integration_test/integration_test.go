package integrationtest_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const BufConnSize = 1024 * 1024

func ContextBufDialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestIntegration(t *testing.T) {
	ctx := context.Background()

	db := NewDB(t)

	apiserverGRPCServer := NewAPIServer(t, ctx, db)
	apiserverBufconn := bufconn.Listen(BufConnSize)
	go func() {
		err := apiserverGRPCServer.Serve(apiserverBufconn)
		if err != nil {
			t.Fatalf("failed to serve apiserver: %v", err)
		}
	}()

	helperGRPCServer := NewHelper(t, ctx)
	helperBufconn := bufconn.Listen(BufConnSize)
	go func() {
		err := helperGRPCServer.Serve(helperBufconn)
		if err != nil {
			t.Fatalf("failed to serve helper: %v", err)
		}
	}()

	apiserverConn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(ContextBufDialer(apiserverBufconn)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	helperConn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(ContextBufDialer(helperBufconn)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	deviceAgentGRPC := NewDeviceAgent(t, ctx, helperConn, apiserverConn)
	deviceAgentConn := bufconn.Listen(BufConnSize)
	go func() {
		err := deviceAgentGRPC.Serve(deviceAgentConn)
		if err != nil {
			t.Fatalf("failed to serve device agent: %v", err)
		}
	}()

	assert.NoError(t, err)
	assert.NotNil(t, apiserverConnForDeviceAgent)
}
