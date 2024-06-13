package kolide

import (
	"encoding/json"
	"time"
)

type DeviceFailure struct {
	ID          uint64                 `json:"id"`
	CheckID     uint64                 `json:"check_id"`
	Value       map[string]interface{} `json:"value"`
	Title       string                 `json:"title"`
	Timestamp   *time.Time             `json:"timestamp"`
	ResolvedAt  *time.Time             `json:"resolved_at"`
	LastUpdated time.Time              `json:"-"`
	Ignored     bool                   `json:"ignored"`
	Check       Check                  `json:"check"`
}

type DeviceFailureWithDevice struct {
	DeviceFailure
	Device Device `json:"device"`
}

type DeviceOwner struct {
	Email string `json:"email"`
}

type Device struct {
	ID              uint64          `json:"id"`
	Name            string          `json:"name"`
	OwnedBy         string          `json:"owned_by"`
	Platform        string          `json:"platform"`
	LastSeenAt      *time.Time      `json:"last_seen_at"`
	FailureCount    int             `json:"failure_count"`
	PrimaryUserName string          `json:"primary_user_name"`
	Serial          string          `json:"serial"`
	AssignedOwner   DeviceOwner     `json:"assigned_owner"`
	Failures        []DeviceFailure `json:"failures"`
}

type Check struct {
	ID          uint64   `json:"id"`
	Tags        []string `json:"tags"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
}

type Pagination struct {
	Next          string `json:"next"`
	NextCursor    string `json:"next_cursor"`
	CurrentCursor string `json:"current_cursor"`
	Count         int    `json:"count"`
}

type PaginatedResponse struct {
	Data       []json.RawMessage `json:"data"`
	Pagination Pagination        `json:"pagination"`
}

type Severity int

const (
	SeverityInfo Severity = iota
	SeverityNotice
	SeverityWarning
	SeverityDanger
	SeverityCritical
)

type SeverityDuration time.Duration

// Devices are allowed to connect this long after a failed check is triggered.
const (
	DurationCritical = 0
	DurationDanger   = time.Hour
	DurationWarning  = time.Hour * 24 * 2
	DurationNotice   = time.Hour * 24 * 7
	DurationUnknown  = time.Hour * 24 * 30
)
