package secretmanager

import (
	"context"
	"fmt"
	"google.golang.org/api/iterator"

	gsecretmanager "cloud.google.com/go/secretmanager/apiv1"
	gsecretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type Secret struct {
	Name string
	Data []byte
}

type SecretManager struct {
	Client  *gsecretmanager.Client
	Project string
}

func New(project string) (*SecretManager, error) {
	client, err := gsecretmanager.NewClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("creating Google Secret Manager client; %w", err)
	}

	return &SecretManager{
		Client:  client,
		Project: project,
	}, nil
}

func (sm *SecretManager) listSecrets() ([]*gsecretmanagerpb.Secret, error) {
	var result []*gsecretmanagerpb.Secret
	ctx := context.Background()
	request := &gsecretmanagerpb.ListSecretsRequest{Parent: fmt.Sprintf("projects/%s", sm.Project)}
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

func (sm *SecretManager) getLatestVersion(secret *gsecretmanagerpb.Secret) (*gsecretmanagerpb.SecretVersion, error) {
	return sm.Client.GetSecretVersion(context.Background(), &gsecretmanagerpb.GetSecretVersionRequest{Name: fmt.Sprintf("%s/versions/latest", secret.Name)})
}

func (sm *SecretManager) accessSecretVersion(secretVersion *gsecretmanagerpb.SecretVersion) ([]byte, error) {
	ctx := context.Background()
	request := &gsecretmanagerpb.AccessSecretVersionRequest{Name: secretVersion.Name}
	accessSecretVersion, err := sm.Client.AccessSecretVersion(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("accessing secret: %w", err)
	}

	return accessSecretVersion.Payload.Data, nil
}

func matchesAllLabels(has, need map[string]string) bool {
	for key, value := range need {
		if secretValue, ok := has[key]; ok {
			if secretValue != value {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func (sm *SecretManager) DisableSecret(name string) error {
	secret, err := sm.Client.GetSecret(
		context.Background(), &gsecretmanagerpb.GetSecretRequest{
			Name: fmt.Sprintf("projects/%s/secrets/%s", sm.Project, name),
		},
	)
	if err != nil {
		return fmt.Errorf("getting secret from project %s: %w", sm.Project, err)
	}

	latestVersion, err := sm.getLatestVersion(secret)
	if err != nil {
		return fmt.Errorf("getting latest secret version: %w", err)
	}

	_, err = sm.Client.DisableSecretVersion(context.Background(), &gsecretmanagerpb.DisableSecretVersionRequest{Name: latestVersion.Name})
	if err != nil {
		return fmt.Errorf("disabling secret version: %w", err)
	}

	return nil
}

func (sm *SecretManager) GetSecret(name string) (*Secret, error) {
	secret, err := sm.Client.GetSecret(
		context.Background(), &gsecretmanagerpb.GetSecretRequest{
			Name: fmt.Sprintf("projects/%s/secrets/%s", sm.Project, name),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("getting secret from project %s: %w", sm.Project, err)
	}

	latestVersion, err := sm.getLatestVersion(secret)
	if err != nil {
		return nil, fmt.Errorf("getting latest secret version: %w", err)
	}

	if latestVersion.State != gsecretmanagerpb.SecretVersion_ENABLED {
		return nil, fmt.Errorf("latest version is not enabled")
	}

	secretData, err := sm.accessSecretVersion(latestVersion)
	if err != nil {
		return nil, fmt.Errorf("accessing secret: %w", err)
	}

	return &Secret{Name: name, Data: secretData}, nil
}

func (sm *SecretManager) GetSecrets(filter map[string]string) ([]*Secret, error) {
	var secrets []*Secret
	googleSecrets, err := sm.listSecrets()
	if err != nil {
		return nil, err
	}

	for _, secret := range googleSecrets {
		if !matchesAllLabels(secret.Labels, filter) {
			continue
		}

		googleVersion, err := sm.getLatestVersion(secret)
		if err != nil {
			return nil, err
		}

		if googleVersion.State != gsecretmanagerpb.SecretVersion_ENABLED {
			continue
		}

		data, err := sm.accessSecretVersion(googleVersion)
		if err != nil {
			return nil, err
		}

		secrets = append(secrets, &Secret{
			Name: secret.Name,
			Data: data,
		})
	}

	return secrets, nil
}
