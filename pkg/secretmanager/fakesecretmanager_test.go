package secretmanager_test

import (
	"context"
	"fmt"
	gsecretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type fakeSecretManager struct {
	gsecretmanagerpb.UnimplementedSecretManagerServiceServer

	Secrets           []*gsecretmanagerpb.Secret
	SecretVersions    []*gsecretmanagerpb.SecretVersion
	SecretVersionData map[string][]byte
}

func (f *fakeSecretManager) addSecretVersion(secretName, versionName string, data []byte, state gsecretmanagerpb.SecretVersion_State) string {
	secretVersionName := fmt.Sprintf("projects/x/secrets/%s/versions/%s", secretName, versionName)
	f.SecretVersions = append(f.SecretVersions, &gsecretmanagerpb.SecretVersion{Name: secretVersionName, State: state})
	f.SecretVersionData[secretName] = data

	return secretVersionName
}
func (f *fakeSecretManager) addSecret(name string, labels map[string]string) string {
	if f.SecretVersionData == nil {
		f.SecretVersionData = make(map[string][]byte)
	}

	secretName := fmt.Sprintf("projects/x/secrets/%s", name)
	secret := &gsecretmanagerpb.Secret{Name: secretName, Labels: labels}
	f.Secrets = append(f.Secrets, secret)

	return secretName
}

func (f *fakeSecretManager) ListSecrets(ctx context.Context, request *gsecretmanagerpb.ListSecretsRequest) (*gsecretmanagerpb.ListSecretsResponse, error) {
	return &gsecretmanagerpb.ListSecretsResponse{
		Secrets: f.Secrets,
	}, nil
}

func (f *fakeSecretManager) GetSecretVersion(ctx context.Context, in *gsecretmanagerpb.GetSecretVersionRequest) (*gsecretmanagerpb.SecretVersion, error) {
	for _, version := range f.SecretVersions {
		if version.Name == in.Name {
			return version, nil
		}
	}

	return nil, fmt.Errorf("no version found")
}

func (f *fakeSecretManager) AccessSecretVersion(ctx context.Context, in *gsecretmanagerpb.AccessSecretVersionRequest) (*gsecretmanagerpb.AccessSecretVersionResponse, error) {
	if f.SecretVersionData == nil {
		return nil, fmt.Errorf("no secret payloads available, call addSecret() first")
	}
	return &gsecretmanagerpb.AccessSecretVersionResponse{
		Name:    in.Name,
		Payload: &gsecretmanagerpb.SecretPayload{Data: f.SecretVersionData[in.Name]},
	}, nil
}
