package kolide_test

import (
	"encoding/json"
	"testing"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalDevice(t *testing.T) {
	raw := `{
		"id": "123",
		"name": "Test-Device",
		"device_type": "Mac",
		"serial": "ABC123",
		"last_authenticated_at": "2025-12-16T09:40:59.052Z",
		"registered_owner_info": {"identifier": "456"}
	}`
	var d kolide.Device
	require.NoError(t, json.Unmarshal([]byte(raw), &d))
	assert.Equal(t, "123", d.ID)
	assert.Equal(t, "Test-Device", d.Name)
	assert.Equal(t, "Mac", d.Platform)
	assert.Equal(t, "ABC123", d.Serial)
	assert.Equal(t, "456", d.OwnerRef.Identifier)
	assert.NotNil(t, d.LastAuthenticatedAt)
}

func TestUnmarshalIssue(t *testing.T) {
	raw := `{
		"id": "100",
		"title": "Test Issue",
		"exempted": true,
		"detected_at": "2025-08-05T08:41:28.624Z",
		"resolved_at": null,
		"last_rechecked_at": "2025-12-28T13:21:16.000Z",
		"device_information": {"identifier": "10001"},
		"check_information": {"identifier": "1"},
		"value": {"key": "value"}
	}`
	var issue kolide.Issue
	require.NoError(t, json.Unmarshal([]byte(raw), &issue))
	assert.Equal(t, "100", issue.ID)
	assert.Equal(t, "Test Issue", issue.Title)
	assert.True(t, issue.Exempted)
	assert.NotNil(t, issue.DetectedAt)
	assert.Nil(t, issue.ResolvedAt)
	assert.NotNil(t, issue.LastRecheckedAt)
	assert.Equal(t, "10001", issue.DeviceRef.Identifier)
	assert.Equal(t, "1", issue.CheckRef.Identifier)
	assert.Equal(t, "value", issue.Value["key"])
}

func TestUnmarshalCheck(t *testing.T) {
	raw := `{
		"id": "1",
		"issue_title": "Test Check",
		"description": "Test description",
		"check_tags": [{"name": "CRITICAL"}, {"name": "DANGER"}]
	}`
	var c kolide.Check
	require.NoError(t, json.Unmarshal([]byte(raw), &c))
	assert.Equal(t, "1", c.ID)
	assert.Equal(t, "Test Check", c.IssueTitle)
	assert.Equal(t, "Test description", c.Description)
	require.Len(t, c.Tags, 2)
	assert.Equal(t, "CRITICAL", c.Tags[0].Name)
	assert.Equal(t, "DANGER", c.Tags[1].Name)
}

func TestUnmarshalPerson(t *testing.T) {
	raw := `{"id": "123", "email": "test@example.com"}`
	var p kolide.Person
	require.NoError(t, json.Unmarshal([]byte(raw), &p))
	assert.Equal(t, "123", p.ID)
	assert.Equal(t, "test@example.com", p.Email)
}

func TestUnmarshalReference(t *testing.T) {
	raw := `{"identifier": "12345"}`
	var ref kolide.Reference
	require.NoError(t, json.Unmarshal([]byte(raw), &ref))
	assert.Equal(t, "12345", ref.Identifier)
}

func TestUnmarshalPagination(t *testing.T) {
	raw := `{
		"next": "https://api.kolide.com/devices?cursor=abc",
		"next_cursor": "abc",
		"current_cursor": "",
		"count": 25
	}`
	var p kolide.Pagination
	require.NoError(t, json.Unmarshal([]byte(raw), &p))
	assert.Equal(t, "https://api.kolide.com/devices?cursor=abc", p.Next)
	assert.Equal(t, "abc", p.NextCursor)
	assert.Equal(t, "", p.CurrentCursor)
	assert.Equal(t, 25, p.Count)
}

func TestUnmarshalPaginatedResponse(t *testing.T) {
	raw := `{
		"data": [{"id": "1"}, {"id": "2"}],
		"pagination": {"next_cursor": "abc", "count": 2}
	}`
	var resp kolide.PaginatedResponse
	require.NoError(t, json.Unmarshal([]byte(raw), &resp))
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, "abc", resp.Pagination.NextCursor)
	assert.Equal(t, 2, resp.Pagination.Count)
}
