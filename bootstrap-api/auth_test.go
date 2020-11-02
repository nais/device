package bootstrap_api_test

import (
	"fmt"
	"github.com/nais/device/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
	"log"
	"sync"
	"testing"
)

/*
func TestGetSecret(t *testing.T) {
	secrets, err := enrollmentTokens.SecretManager.ListSecrets()
	assert.NoError(t, err)
	for _, secret := range secrets {
		fmt.Printf("%s: %s\n", secret.Secret.Name, string(secret.SecretVersions.Latest().Data))
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
*/
