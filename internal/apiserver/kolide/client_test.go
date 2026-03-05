package kolide_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*httptest.Server, map[string][]byte) {
	testdata := map[string][]byte{}

	files := []string{"devices", "issues", "checks", "people"}
	for _, f := range files {
		data, err := os.ReadFile("testdata/" + f + ".json")
		require.NoError(t, err)
		testdata[f] = data
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/devices":
			w.Write(testdata["devices"])
		case "/issues":
			w.Write(testdata["issues"])
		case "/checks":
			w.Write(testdata["checks"])
		case "/people":
			w.Write(testdata["people"])
		case "/devices/10001/open_issues":
			w.Write(testdata["issues"])
		default:
			t.Errorf("unexpected request to %v", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return server, testdata
}

func TestClient_GetDevices(t *testing.T) {
	ctx := context.Background()
	server, _ := setupTestServer(t)
	defer server.Close()

	client := kolide.New("token", logrus.New(), kolide.WithBaseURL(server.URL))

	devices, err := client.GetDevices(ctx)
	require.NoError(t, err)
	require.Len(t, devices, 4)

	t.Run("first device parsed correctly", func(t *testing.T) {
		d := devices[0]
		assert.Equal(t, "10001", d.ID)
		assert.Equal(t, "MacBook-Pro-001", d.Name)
		assert.Equal(t, "darwin", d.Platform) // converted from "Mac"
		assert.Equal(t, "XXXX1111AAAA", d.Serial)
		assert.Equal(t, "44200", d.OwnerRef.Identifier)
		assert.Equal(t, "john.doe@example.com", d.Owner.Email)
		assert.Equal(t, "44200", d.Owner.ID)
	})

	t.Run("Windows device platform converted", func(t *testing.T) {
		d := devices[2]
		assert.Equal(t, "10003", d.ID)
		assert.Equal(t, "windows", d.Platform) // converted from "Windows"
	})

	t.Run("unregistered device has empty owner", func(t *testing.T) {
		d := devices[3]
		assert.Equal(t, "10004", d.ID)
		assert.Equal(t, "", d.OwnerRef.Identifier)
		assert.Equal(t, "", d.Owner.Email)
	})
}

func TestClient_GetPeople(t *testing.T) {
	ctx := context.Background()
	server, _ := setupTestServer(t)
	defer server.Close()

	client := kolide.New("token", logrus.New(), kolide.WithBaseURL(server.URL))

	people, err := client.GetPeople(ctx)
	require.NoError(t, err)
	require.Len(t, people, 4)

	t.Run("person keyed by id", func(t *testing.T) {
		p, ok := people["44200"]
		require.True(t, ok)
		assert.Equal(t, "44200", p.ID)
		assert.Equal(t, "john.doe@example.com", p.Email)
	})

	t.Run("all expected people present", func(t *testing.T) {
		assert.Contains(t, people, "44200")
		assert.Contains(t, people, "44794")
		assert.Contains(t, people, "45037")
		assert.Contains(t, people, "47221")
	})
}

func TestClient_GetIssues(t *testing.T) {
	ctx := context.Background()
	server, _ := setupTestServer(t)
	defer server.Close()

	client := kolide.New("token", logrus.New(), kolide.WithBaseURL(server.URL))

	issues, err := client.GetIssues(ctx)
	require.NoError(t, err)
	require.Len(t, issues, 3)

	t.Run("first issue parsed correctly", func(t *testing.T) {
		issue := issues[0]
		assert.Equal(t, "100001", issue.ID)
		assert.Equal(t, "Bluetooth Sharing Is Not Disabled", issue.Title)
		assert.Equal(t, false, issue.Exempted)
		assert.Equal(t, "10001", issue.DeviceRef.Identifier)
		assert.Equal(t, "1", issue.CheckRef.Identifier)
	})

	t.Run("SSH key issue parsed correctly", func(t *testing.T) {
		issue := issues[1]
		assert.Equal(t, "100002", issue.ID)
		assert.Equal(t, "Unencrypted SSH Key Detected", issue.Title)
		assert.Equal(t, "3", issue.CheckRef.Identifier)
	})

	t.Run("all issues have timestamps", func(t *testing.T) {
		for _, issue := range issues {
			assert.NotNil(t, issue.DetectedAt)
			assert.NotNil(t, issue.LastRecheckedAt)
		}
	})
}

func TestClient_GetChecks(t *testing.T) {
	ctx := context.Background()
	server, _ := setupTestServer(t)
	defer server.Close()

	client := kolide.New("token", logrus.New(), kolide.WithBaseURL(server.URL))

	checks, err := client.GetChecks(ctx)
	require.NoError(t, err)
	require.Len(t, checks, 2)

	t.Run("bluetooth check parsed correctly", func(t *testing.T) {
		c := checks[0]
		assert.Equal(t, "1", c.ID)
		assert.Equal(t, "Bluetooth Sharing Is Not Disabled", c.IssueTitle)
		assert.Equal(t, "This check ensures the macOS Sharing Setting is disabled.", c.Description)
		require.Len(t, c.Tags, 1)
		assert.Equal(t, "CRITICAL", c.Tags[0].Name)
	})

	t.Run("SSH check parsed correctly", func(t *testing.T) {
		c := checks[1]
		assert.Equal(t, "3", c.ID)
		assert.Equal(t, "Unencrypted SSH Key Detected", c.IssueTitle)
		require.Len(t, c.Tags, 1)
		assert.Equal(t, "DANGER", c.Tags[0].Name)
	})
}

func TestClient_GetDeviceIssues(t *testing.T) {
	ctx := context.Background()
	server, _ := setupTestServer(t)
	defer server.Close()

	client := kolide.New("token", logrus.New(), kolide.WithBaseURL(server.URL))

	issues, err := client.GetDeviceIssues(ctx, "10001")
	require.NoError(t, err)
	require.Len(t, issues, 3)

	t.Run("issues returned for device", func(t *testing.T) {
		assert.Equal(t, "100001", issues[0].ID)
		assert.Equal(t, "Bluetooth Sharing Is Not Disabled", issues[0].Title)
	})
}

func TestClient_EmptyResponse(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data": [], "pagination": {"count": 0}}`))
	}))
	defer server.Close()

	client := kolide.New("token", logrus.New(), kolide.WithBaseURL(server.URL))

	t.Run("empty devices", func(t *testing.T) {
		devices, err := client.GetDevices(ctx)
		require.NoError(t, err)
		assert.Len(t, devices, 0)
	})

	t.Run("empty issues", func(t *testing.T) {
		issues, err := client.GetIssues(ctx)
		require.NoError(t, err)
		assert.Len(t, issues, 0)
	})

	t.Run("empty checks", func(t *testing.T) {
		checks, err := client.GetChecks(ctx)
		require.NoError(t, err)
		assert.Len(t, checks, 0)
	})
}

func TestClient_Pagination(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if r.URL.Path == "/people" {
			w.Write([]byte(`{"data": [], "pagination": {"count": 0}}`))
			return
		}

		cursor := r.URL.Query().Get("cursor")

		switch cursor {
		case "":
			w.Write([]byte(`{
				"data": [{"id": "1", "name": "Device-1", "device_type": "Mac", "serial": "AAA", "registered_owner_info": {"identifier": ""}}],
				"pagination": {"next_cursor": "page2", "count": 1}
			}`))
		case "page2":
			w.Write([]byte(`{
				"data": [{"id": "2", "name": "Device-2", "device_type": "Mac", "serial": "BBB", "registered_owner_info": {"identifier": ""}}],
				"pagination": {"next_cursor": "", "count": 1}
			}`))
		}
	}))
	defer server.Close()

	client := kolide.New("token", logrus.New(), kolide.WithBaseURL(server.URL))

	devices, err := client.GetDevices(ctx)
	require.NoError(t, err)
	assert.Len(t, devices, 2)
	assert.Equal(t, 3, callCount) // 2 pages of devices + 1 people call
	assert.Equal(t, "1", devices[0].ID)
	assert.Equal(t, "2", devices[1].ID)
}

func TestClient_GetDevice(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/devices/10001":
			w.Write([]byte(`{
				"id": "10001",
				"name": "MacBook-Pro-001",
				"device_type": "Mac",
				"serial": "XXXX1111AAAA",
				"registered_owner_info": {"identifier": "44200"}
			}`))
		case "/people/44200":
			w.Write([]byte(`{"id": "44200", "email": "john.doe@example.com"}`))
		default:
			t.Errorf("unexpected request to %v", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := kolide.New("token", logrus.New(), kolide.WithBaseURL(server.URL))

	t.Run("device with owner resolved", func(t *testing.T) {
		d, err := client.GetDevice(ctx, "10001")
		require.NoError(t, err)
		assert.Equal(t, "10001", d.ID)
		assert.Equal(t, "MacBook-Pro-001", d.Name)
		assert.Equal(t, "darwin", d.Platform) // converted from "Mac"
		assert.Equal(t, "XXXX1111AAAA", d.Serial)
		assert.Equal(t, "44200", d.OwnerRef.Identifier)
		assert.Equal(t, "44200", d.Owner.ID)
		assert.Equal(t, "john.doe@example.com", d.Owner.Email)
	})

	t.Run("device with no owner has empty Owner", func(t *testing.T) {
		noOwnerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/devices/10004":
				w.Write([]byte(`{
					"id": "10004",
					"name": "MacBook-Pro-003",
					"device_type": "Mac",
					"serial": "XXXX4444DDDD",
					"registered_owner_info": {"identifier": ""}
				}`))
			default:
				t.Errorf("unexpected request to %v", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer noOwnerServer.Close()

		noOwnerClient := kolide.New("token", logrus.New(), kolide.WithBaseURL(noOwnerServer.URL))
		d, err := noOwnerClient.GetDevice(ctx, "10004")
		require.NoError(t, err)
		assert.Equal(t, "10004", d.ID)
		assert.Equal(t, "", d.OwnerRef.Identifier)
		assert.Equal(t, "", d.Owner.Email)
	})
}
