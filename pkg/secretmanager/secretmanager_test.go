package secretmanager_test

import (
	"context"
	"net"

	"github.com/nais/device/pkg/secretmanager"
	"github.com/stretchr/testify/assert"

	gsecretmanager "cloud.google.com/go/secretmanager/apiv1"
	"google.golang.org/api/option"
	gsecretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"google.golang.org/grpc"

	"testing"
)

func TestGetSecrets(t *testing.T) {
	fsm := &fakeSecretManager{Project: "x"}
	sm, err := setup(t, fsm)
	assert.NoError(t, err)

	fsm.addSecret("one", map[string]string{"label1": "value1", "label2": "value2"})
	fsm.addSecret("two", map[string]string{"label2": "value2"})
	fsm.addSecret("three", map[string]string{"label3": "value3"})

	fsm.addSecretVersion("one", "latest", []byte("1"), gsecretmanagerpb.SecretVersion_ENABLED)
	fsm.addSecretVersion("two", "latest", []byte("2"), gsecretmanagerpb.SecretVersion_ENABLED)
	fsm.addSecretVersion("three", "latest", []byte("3"), gsecretmanagerpb.SecretVersion_DISABLED)

	t.Run("nil filter returns all enabled secrets", func(t *testing.T) {
		secrets, err := sm.GetSecrets(nil)
		assert.NoError(t, err)
		assert.Len(t, secrets, 2)
	})

	t.Run("filter works", func(t *testing.T) {
		secrets, err := sm.GetSecrets(map[string]string{"label1": "value1", "label2": "value2"})
		assert.NoError(t, err)
		assert.Len(t, secrets, 1)
	})

	t.Run("get secret", func(t *testing.T) {
		secret, err := sm.GetSecret("one")
		assert.NoError(t, err)
		assert.Equal(t, []byte("1"), secret.Data)
	})

	t.Run("get disabled secret", func(t *testing.T) {
		_, err := sm.GetSecret("three")
		assert.Error(t, err)
	})

}

func setup(t *testing.T, fsm *fakeSecretManager) (*secretmanager.SecretManager, error) {
	ctx := context.Background()

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	gserver := grpc.NewServer()
	gsecretmanagerpb.RegisterSecretManagerServiceServer(gserver, fsm)
	fakeServerAddress := l.Addr().String()

	go func() {
		if err := gserver.Serve(l); err != nil {
			panic(err)
		}
	}()

	client, err := gsecretmanager.NewClient(ctx, option.WithEndpoint(fakeServerAddress), option.WithoutAuthentication(), option.WithGRPCDialOption(grpc.WithInsecure()))
	if err != nil {
		t.Fatal(err)
	}

	sm := &secretmanager.SecretManager{
		Client:  client,
		Project: fsm.Project,
	}

	return sm, err
}
