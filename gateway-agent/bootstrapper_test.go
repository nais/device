package gateway_agent_test

import (
	g "github.com/nais/device/gateway-agent"
	"github.com/nais/device/pkg/secretmanager"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

type FakeSecretManager struct {
	secrets []*secretmanager.Secret
}

func (sm *FakeSecretManager) GetSecrets(_ map[string]string) ([]*secretmanager.Secret, error) {
	return sm.secrets, nil
}

func TestGetBootstrapConfig(t *testing.T) {
	sm := FakeSecretManager{secrets: []*secretmanager.Secret{{Name: "secret", Data: []byte("s3cr3t")}}}
	t.Run("returns existing bootstrapconfig if present", func(t *testing.T) {
		f, err := ioutil.TempFile(os.TempDir(), "test")
		assert.NoError(t, err)
		defer os.Remove(f.Name())
		defer f.Close()

		deviceIP := "10.255.240.31"
		f.WriteString(`{"deviceIP":"` + deviceIP + `"}`)
		cfg := &g.Config{BootstrapConfigPath: f.Name()}
		bootstrapper := g.Bootstrapper{SecretManager: &sm, Config: cfg}
		config, err := bootstrapper.GetBootstrapConfig()
		assert.Equal(t, deviceIP, config.DeviceIP)
	})

}
