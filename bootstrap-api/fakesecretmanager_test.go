package bootstrap_api_test

import (
	"context"

	gsecretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type fakeSecretManager struct {
	gsecretmanagerpb.UnimplementedSecretManagerServiceServer
}

func (f *fakeSecretManager) ListSecrets (ctx context.Context, request *gsecretmanagerpb.ListSecretsRequest) (*gsecretmanagerpb.ListSecretsResponse, error) {
	response := &gsecretmanagerpb.ListSecretsResponse{

	}

	return response, nil
}
