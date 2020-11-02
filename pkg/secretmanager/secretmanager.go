package secretmanager

import (
	"context"
	"fmt"
	"google.golang.org/api/iterator"

	gsecretmanager "cloud.google.com/go/secretmanager/apiv1"
	gsecretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type SecretVersion struct {
	Data    []byte
	Version *gsecretmanagerpb.SecretVersion
}

type Secret struct {
	Secret         *gsecretmanagerpb.Secret
	SecretVersions SecretVersions
}

type SecretVersions []*SecretVersion

type SecretManager struct {
	Client *gsecretmanager.Client
}



func (versions SecretVersions) Latest() *SecretVersion {
	var latest *SecretVersion
	for _, version := range versions {
		if latest == nil {
			latest = version
		}

		if version.Version.CreateTime.AsTime().After(latest.Version.CreateTime.AsTime()) {
			latest = version
		}
	}

	return latest
}

func New() (*SecretManager, error) {
	client, err := gsecretmanager.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("creating Google Secret Manager client; %w", err)
	}

	return &SecretManager{Client: client}, nil
}

func (sm *SecretManager) GetSecrets() ([]*gsecretmanagerpb.Secret, error) {
	var result []*gsecretmanagerpb.Secret
	ctx := context.Background()
	request := &gsecretmanagerpb.ListSecretsRequest{Parent: "projects/nais-device"}
	secrets := sm.Client.ListSecrets(ctx, request)
	for {
		secret, err := secrets.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("getting secret: %w", err)
		}

		result = append(result, secret)
	}

	return result, nil
}

func (sm *SecretManager) GetSecretVersions(secret *gsecretmanagerpb.Secret) ([]*gsecretmanagerpb.SecretVersion, error) {
	var result []*gsecretmanagerpb.SecretVersion
	ctx := context.Background()
	request := &gsecretmanagerpb.ListSecretVersionsRequest{Parent: secret.Name}
	secretVersion := sm.Client.ListSecretVersions(ctx, request)

	for {
		secret, err := secretVersion.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("getting secret: %w", err)
		}

		result = append(result, secret)
	}

	return result, nil
}

func (sm *SecretManager) GetSecretVersionData(secretVersion *gsecretmanagerpb.SecretVersion) ([]byte, error) {
	ctx := context.Background()
	request := &gsecretmanagerpb.AccessSecretVersionRequest{Name: secretVersion.Name}
	accessSecretVersion, err := sm.Client.AccessSecretVersion(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("accessing secret: %w", err)
	}

	return accessSecretVersion.Payload.Data, nil
}

func (sm *SecretManager) ListSecrets(filter map[string]string) ([]*Secret, error) {
	var secrets []*Secret
	googleSecrets, err := sm.GetSecrets()
	if err != nil {
		return nil, err
	}

	secretLoop:
	for _, secret := range googleSecrets {
		for key, value := range filter {
			if secretValue, ok := secret.Labels[key]; ok {
				if secretValue != value {
					continue secretLoop
				}
			} else {
				continue secretLoop
			}
		}

		googleVersion, err := sm.GetSecretVersions(secret)
		if err != nil {
			return nil, err
		}

		var secretVersions []*SecretVersion

		for _, version := range googleVersion {

			if version.State != gsecretmanagerpb.SecretVersion_ENABLED {
				continue
			}

			data, err := sm.GetSecretVersionData(version)
			if err != nil {
				return nil, err
			}

			secretVersions = append(secretVersions, &SecretVersion{
				Data:    data,
				Version: version,
			})
		}

		if len(secretVersions) > 0 {
			secrets = append(secrets, &Secret{
				Secret:         secret,
				SecretVersions: secretVersions,
			})
		}
	}

	return secrets, nil
}
