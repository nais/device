package bootstrap_api_test

import (
	"context"
	"fmt"
	"net"

	gsecretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/nais/device/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
	gsecretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"google.golang.org/grpc"

	"log"
	"sync"
	"testing"
)


func TestGetSecret(t *testing.T) {
	// t.SkipNow()

	filter := map[string]string {"type": "enrollment-token"}
	secrets, err := enrollmentTokens.SecretManager.ListSecrets(filter)
	assert.NoError(t, err)
	for _, secret := range secrets {
		fmt.Printf("%s: %s\n", secret.Secret.Name, string(secret.SecretVersions.Latest().Data))

	}


}

func TestListSecrets(t *testing.T) {
	ctx := context.Background()
	req := &gsecretmanagerpb.ListSecretsRequest{
		Parent: fmt.Sprintf("projects/%s", "test"),
	}

	fakeSecretManagerServer := &fakeSecretManager{}
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	gserver := grpc.NewServer()
	gsecretmanagerpb.RegisterSecretManagerServiceServer(gserver, fakeSecretManagerServer)
	fakeServerAddress := l.Addr().String()

	go func() {
		if err := gserver.Serve(l); err != nil { panic(err)}
	}()

	client, err := gsecretmanager.NewClient(ctx, option.WithEndpoint(fakeServerAddress), option.WithoutAuthentication(), option.WithGRPCDialOption(grpc.WithInsecure()))
	if err != nil {
		t.Fatal(err)
	}

	

}

var enrollmentTokens EnrollmentTokens

type EnrollmentTokens struct {
	Active        map[string]string
	ActiveLock    sync.Mutex
	SecretManager *secretmanager.SecretManager
}

func init() {
	var err error
	enrollmentTokens.SecretManager, err = secretmanager.New()

	if err != nil {
		log.Fatalf("Instantiating SecretManager: %v", err)
	}
}

