package kolide_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalDevice(t *testing.T) {
	raw := `{
  "id": "10001",
  "name": "MacBook-Pro-001",
  "registered_at": "2025-12-16T09:40:59.052Z",
  "last_authenticated_at": null,
  "registered_owner_info": {
    "identifier": "44200",
    "link": "https://api.kolide.com/people/44200"
  },
  "operating_system": "macOS 15.7.3 Sequoia",
  "hardware_model": "MacBook Pro (15-inch, 2018)",
  "serial": "XXXX1111AAAA",
  "hardware_uuid": "00000000-1111-2222-3333-444444444444",
  "note": null,
  "auth_state": "Good",
  "will_block_at": null,
  "product_image_url": "https://example.com/macbook-pro.png",
  "auth_configuration": {
    "device_id": "10001",
    "authentication_mode": "anyone",
    "person_groups": []
  },
  "device_type": "Mac",
  "form_factor": "computer"
}`
	device := kolide.Device{}
	assert.NoError(t, json.Unmarshal([]byte(raw), &device))
	assert.Equal(t, "10001", device.ID)
	assert.Equal(t, "MacBook-Pro-001", device.Name)
	assert.Equal(t, "Mac", device.Platform)
	assert.Equal(t, "XXXX1111AAAA", device.Serial)
	assert.Equal(t, "44200", device.OwnerRef.Identifier)
}

func TestUnmarshalIssue(t *testing.T) {
	raw := `{
  "id": "100001",
  "issue_key": "username",
  "issue_value": "johndoe",
  "title": "Bluetooth Sharing Is Not Disabled",
  "value": {
    "path": "/Users/johndoe/Library/Preferences/ByHost/com.apple.Bluetooth.ABC123.plist",
    "username": "johndoe",
    "logged_in": "1",
    "current_os_version": "15.7.3",
    "KOLIDE_CHECK_STATUS": "FAIL",
    "user_bluetooth_sharing_enabled": "1"
  },
  "exempted": false,
  "resolved_at": null,
  "detected_at": "2025-08-05T08:41:28.624Z",
  "blocks_device_at": "2025-12-28T21:59:45.789Z",
  "device_information": {
    "identifier": "10001",
    "link": "https://api.kolide.com/devices/10001"
  },
  "check_information": {
    "identifier": "1",
    "link": "https://api.kolide.com/checks/1"
  },
  "last_rechecked_at": "2025-12-28T13:21:16.000Z"
}`
	issue := kolide.Issue{}
	assert.NoError(t, json.Unmarshal([]byte(raw), &issue))
	assert.Equal(t, "100001", issue.ID)
	assert.Equal(t, "Bluetooth Sharing Is Not Disabled", issue.Title)
	assert.Equal(t, false, issue.Exempted)
	assert.Nil(t, issue.ResolvedAt)
	assert.Equal(t, "10001", issue.DeviceRef.Identifier)
	assert.Equal(t, "1", issue.CheckRef.Identifier)
	assert.Equal(t, "johndoe", issue.Value["username"])

	expectedDetectedAt := time.Date(2025, 8, 5, 8, 41, 28, 624000000, time.UTC)
	assert.True(t, expectedDetectedAt.Equal(*issue.DetectedAt))

	expectedLastRechecked := time.Date(2025, 12, 28, 13, 21, 16, 0, time.UTC)
	assert.True(t, expectedLastRechecked.Equal(*issue.LastRecheckedAt))
}

func TestUnmarshalCheck(t *testing.T) {
	raw := `{
  "id": "1",
  "name": "macOS Sharing - Require Bluetooth Sharing to Be Disabled",
  "slug": "macos_bluetooth_sharing",
  "compatible_platforms": ["darwin"],
  "description": "This check ensures the macOS Sharing Setting is disabled.",
  "topics": ["sharing-preferences"],
  "issue_title": "Bluetooth Sharing Is Not Disabled",
  "check_tags": [
    {
      "id": "3388",
      "name": "CRITICAL",
      "description": "Checks that produce failures which should be resolved immediately.",
      "color": "rgba(255, 40, 122, 1)"
    }
  ],
  "kolide_provided": true,
  "type": "osquery"
}`
	check := kolide.Check{}
	assert.NoError(t, json.Unmarshal([]byte(raw), &check))
	assert.Equal(t, "1", check.ID)
	assert.Equal(t, "Bluetooth Sharing Is Not Disabled", check.IssueTitle)
	assert.Equal(t, "This check ensures the macOS Sharing Setting is disabled.", check.Description)
	assert.Len(t, check.Tags, 1)
	assert.Equal(t, "CRITICAL", check.Tags[0].Name)
}

func TestUnmarshalPerson(t *testing.T) {
	raw := `{
  "id": "44200",
  "name": "Doe, John",
  "email": "john.doe@example.com",
  "created_at": "2020-03-10T08:22:46.793Z",
  "last_authenticated_at": null,
  "has_registered_device": true,
  "usernames": [
    "John.Doe@example.com"
  ]
}`
	person := kolide.Person{}
	assert.NoError(t, json.Unmarshal([]byte(raw), &person))
	assert.Equal(t, "44200", person.ID)
	assert.Equal(t, "john.doe@example.com", person.Email)
}

func TestUnmarshalPaginatedResponse(t *testing.T) {
	data, err := os.ReadFile("testdata/devices.json")
	require.NoError(t, err)

	var response kolide.PaginatedResponse
	assert.NoError(t, json.Unmarshal(data, &response))
	assert.Len(t, response.Data, 4)
	assert.Equal(t, 4, response.Pagination.Count)
	assert.Equal(t, "", response.Pagination.Next)
	assert.Equal(t, "", response.Pagination.NextCursor)
}

func TestUnmarshalExternalInfo(t *testing.T) {
	raw := `{
  "identifier": "12345",
  "link": "https://api.kolide.com/resource/12345"
}`
	info := kolide.Reference{}
	assert.NoError(t, json.Unmarshal([]byte(raw), &info))
	assert.Equal(t, "12345", info.Identifier)
}
