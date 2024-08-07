package kolide_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
			case "/failures/open":
				fmt.Fprintf(w, `{}`)
			default:
				t.Errorf("unexpected request to %v", r.URL.Path)
			}
		}))

		client := kolide.New("token", logrus.New(), kolide.WithBaseUrl(s.URL))
		devices, err := client.GetDevices(ctx)
		assert.NoError(t, err)
		assert.Len(t, devices, 0)
	})

	t.Run("get all kolide data", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/devices":
				fmt.Fprintf(w, `
{
  "data": [
    {
      "id": 1,
      "name": "LAPTOP-ASD123",
      "platform": "ubuntu",
      "last_seen_at": "2024-01-01T00:00:00.000Z",
      "issue_count": 1,
      "resolved_issue_count": 0,
      "failure_count": 1,
      "resolved_failure_count": 0,
      "serial": "TEST-SERIAL",
      "assigned_owner": {
        "id": 1,
        "owner_type": "Person",
        "name": "Test Testesen",
        "email": "test@example.com"
      }
    }
  ],
  "pagination": {
    "next": "",
    "next_cursor": "",
    "current_cursor": "",
    "count": 1
  }
}
				`)
			case "/checks/1":
				fmt.Fprintf(w, `
{
  "id": 1,
  "failing_device_count": 1,
  "display_name": "test check display name",
  "description": "test check description",
  "tags": [
    "CRITICAL"
  ]
}
				`)
			case "/failures/open":
				fmt.Fprintf(w, `
{
  "data": [
    {
      "id": 1,
      "check_id": 1,
      "title": "test failure title",
      "ignored": false,
      "resolved_at": null,
      "timestamp": "1970-01-01T00:00:00.000Z",
      "device": {
        "id": 1,
        "platform": "ubuntu",
        "serial": "TEST-SERIAL"
      }
    }
  ],
  "pagination": {
    "next": "",
    "next_cursor": "",
    "current_cursor": "",
    "count": 1
  }
}
				`)
			default:
				t.Errorf("unexpected request to %v", r.URL.Path)
			}
		}))

		client := kolide.New("token", logrus.New(), kolide.WithBaseUrl(s.URL))

		devices, err := client.GetDevices(ctx)
		assert.NoError(t, err)
		issues, err := client.GetIssues(ctx)
		assert.NoError(t, err)
		assert.Len(t, issues, 1)
		assert.Equal(t, "test failure title", issues[0].Title)
		assert.Equal(t, "2024-01-01 00:00:00 +0000 UTC", devices[0].LastSeenAt.String())
		assert.Equal(t, int64(1), devices[0].ID)
	})
}
