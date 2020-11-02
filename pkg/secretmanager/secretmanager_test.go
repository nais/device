package secretmanager_test

import (
	"context"
	"fmt"
	"github.com/nais/device/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
	"net"

	gsecretmanager "cloud.google.com/go/secretmanager/apiv1"
	"google.golang.org/api/option"
	gsecretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"google.golang.org/grpc"

	"testing"
)

func TestListSecrets(t *testing.T) {
	testdata := []*secretmanager.Secret{{
		GoogleSecret: &gsecretmanagerpb.Secret{Name: "x",
			Labels: map[string]string{"foo": "bar"}},
		SecretVersions: secretmanager.SecretVersions{{
			Data: []byte("hemmelig"),
			GoogleVersion: &gsecretmanagerpb.SecretVersion{
				State: gsecretmanagerpb.SecretVersion_ENABLED,
			},
		}},
	}}

	sm, err := setup(t, &fakeSecretManager{Secrets: testdata})

	filter := map[string]string{"foo": "bar"}
	secrets, err := sm.ListSecrets(filter)
	assert.NoError(t, err)

	for _, secret := range secrets {
		fmt.Println(string(secret.SecretVersions.Latest().Data))
	}
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
		Client: client,
	}

	return sm, err
}

//var enrollmentTokens EnrollmentTokens
//
//type EnrollmentTokens struct {
//	Active        map[string]string
//	ActiveLock    sync.Mutex
//	SecretManager *secretmanager.SecretManager
//}

//func init() {
//	var err error
//	enrollmentTokens.SecretManager, err = New()
//
//	if err != nil {
//		log.Fatalf("Instantiating SecretManager: %v", err)
//	}
//}
