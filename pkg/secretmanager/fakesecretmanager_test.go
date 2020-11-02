package secretmanager_test

import (
	"context"
	"fmt"
	"github.com/nais/device/pkg/secretmanager"
	gsecretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type fakeSecretManager struct {
	gsecretmanagerpb.UnimplementedSecretManagerServiceServer
	Secrets []*secretmanager.Secret
}

func (f *fakeSecretManager) ListSecrets(ctx context.Context, request *gsecretmanagerpb.ListSecretsRequest) (*gsecretmanagerpb.ListSecretsResponse, error) {
	response := &gsecretmanagerpb.ListSecretsResponse{
		Secrets: secrets(f.Secrets),
	}

	return response, nil
}

func (f *fakeSecretManager) ListSecretVersions(ctx context.Context, in *gsecretmanagerpb.ListSecretVersionsRequest) (*gsecretmanagerpb.ListSecretVersionsResponse, error) {
	secret := getByName(f.Secrets, in.Parent)
	if secret == nil {
		return nil, fmt.Errorf("no secret found")
	}

	return &gsecretmanagerpb.ListSecretVersionsResponse{
		Versions: versions(secret.SecretVersions),
	}, nil
}

func (f *fakeSecretManager) AccessSecretVersion(ctx context.Context, in *gsecretmanagerpb.AccessSecretVersionRequest) (*gsecretmanagerpb.AccessSecretVersionResponse, error) {
	v := getVersion(f.Secrets, in.Name)
	if v == nil {
		return nil, fmt.Errorf("no secret version found for secret: %s", in.Name)
	}
	return &gsecretmanagerpb.AccessSecretVersionResponse{
		Name:    v.GoogleVersion.Name,
		Payload: &gsecretmanagerpb.SecretPayload{Data: v.Data},
	}, nil
}

func getByName(secrets []*secretmanager.Secret, name string) *secretmanager.Secret {
	for _, secret := range secrets {
		if secret.GoogleSecret.Name == name {
			return secret
		}
	}

	return nil
}

func getVersion(secrets []*secretmanager.Secret, versionPath string) *secretmanager.SecretVersion {
	for _, secret := range secrets {
		for _, version := range secret.SecretVersions {
			if version.GoogleVersion.Name == versionPath {
				return version
			}
		}
	}

	return nil
}

func versions(versions []*secretmanager.SecretVersion) (googleVersions []*gsecretmanagerpb.SecretVersion) {
	for _, version := range versions {
		googleVersions = append(googleVersions, version.GoogleVersion)
	}

	return
}

func secrets(secrets []*secretmanager.Secret) (googleSecrets []*gsecretmanagerpb.Secret) {
	for _, secret := range secrets {
		googleSecrets = append(googleSecrets, secret.GoogleSecret)
	}

	return
}
