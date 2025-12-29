package kolide_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	ctx := context.Background()

	// Create a new client

	t.Run("smoke screen client test", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/devices":
				fmt.Fprintf(w, `{}`)
			case "/checks":
				fmt.Fprintf(w, `{}`)
			case "/issues":
				fmt.Fprintf(w, `{}`)
			case "/people":
				fmt.Fprintf(w, `{}`)
			default:
				t.Errorf("unexpected request to %v", r.URL.Path)
			}
		}))

		client := kolide.New("token", logrus.New(), kolide.WithBaseURL(s.URL))
		devices, err := client.GetDevices(ctx)
		assert.NoError(t, err)
		assert.Len(t, devices, 0)
	})

	t.Run("get all kolide data", func(t *testing.T) {
		devicesData, err := os.ReadFile("testdata/devices.json")
		require.NoError(t, err)
		issuesData, err := os.ReadFile("testdata/issues.json")
		require.NoError(t, err)
		peopleData, err := os.ReadFile("testdata/people.json")
		require.NoError(t, err)
		checksData, err := os.ReadFile("testdata/checks.json")
		require.NoError(t, err)

		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/devices":
				w.Write(devicesData)
			case "/issues":
				w.Write(issuesData)
			case "/people":
				w.Write(peopleData)
			case "/checks":
				w.Write(checksData)
			default:
				t.Errorf("unexpected request to %v", r.URL.Path)
			}
		}))

		client := kolide.New("token", logrus.New(), kolide.WithBaseURL(s.URL))

		devices, err := client.GetDevices(ctx)
		assert.NoError(t, err)
		issues, err := client.GetIssues(ctx)
		assert.NoError(t, err)
		assert.Len(t, issues, 3)
		assert.Equal(t, "Bluetooth Sharing Is Not Disabled", issues[0].Title)
		assert.Len(t, devices, 4)
		assert.Equal(t, "10001", devices[0].ID)
		assert.Equal(t, "MacBook-Pro-001", devices[0].Name)
	})
}
