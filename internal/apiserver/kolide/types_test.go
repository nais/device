package kolide_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalDevice(t *testing.T) {
	raw := `{
  "id": 1,
  "name": "device-name",
  "owned_by": "organization",
  "privacy": "details_visible",
  "platform": "nixos",
  "enrolled_at": "2024-04-30T08:10:57.967Z",
  "last_seen_at": "2024-06-26T10:12:00.000Z",
  "operating_system": "NixOS 24.11 (Vicuna) ",
  "issue_count": 0,
  "resolved_issue_count": 1,
  "failure_count": 0,
  "resolved_failure_count": 1,
  "hardware_model": "Precision 5470",
  "hardware_vendor": "Dell",
  "launcher_version": "1.8.1-2-gedc91c6",
  "osquery_version": "5.12.2",
  "serial": "device-serial",
  "hardware_uuid": "d115e4dd-aa4f-4ae1-9a23-88526addeafa",
  "assigned_owner": {
    "id": 1,
    "owner_type": "Person",
    "name": "Name Namesen",
    "email": "name@example.com"
  },
  "kolide_mdm": null,
  "note": null,
  "note_html": null,
  "operating_system_details": {
    "device_id": 1,
    "platform": "nixos",
    "name": "NixOS",
    "codename": "vicuna",
    "version": "24.11 (Vicuna)",
    "build": "24.11.20240622.a71e967",
    "major_version": 24,
    "minor_version": 11,
    "patch_version": 0,
    "ubr": null,
    "release_id": null
  },
  "remote_ip": "13.37.13.37",
  "location": null,
  "product_image_url": "https://assets1.kolide.com/assets/inventory/devices/linux-929081ebf0aa2a3cfbb48ae3ffbb0db58aee09ac.png"
}`
	device := kolide.Device{}
	assert.NoError(t, json.Unmarshal([]byte(raw), &device))
	assert.Equal(t, uint64(1), device.ID)
	assert.Equal(t, "device-name", device.Name)
	assert.Equal(t, "organization", device.OwnedBy)
	assert.Equal(t, "nixos", device.Platform)
	assert.Equal(t, "device-serial", device.Serial)
	assert.Equal(t, "name@example.com", device.AssignedOwner.Email)
	// "last_seen_at": "2024-06-26T10:12:00.000Z",
	assert.True(t, time.Date(2024, 6, 26, 10, 12, 0, 0, time.UTC).Equal(*device.LastSeenAt))
}
