package kolide_client

import (
	"encoding/json"
	"time"
)

type DeviceFailure struct {
	Id         int                    `json:"id"`
	CheckId    int                    `json:"check_id"`
	Value      map[string]interface{} `json:"value"`
	Title      string                 `json:"title"`
	Timestamp  *time.Time              `json:"timestamp"`
	ResolvedAt *time.Time              `json:"resolved_at"`
	Ignored    bool                   `json:"ignored"`
	Check      *Check                 `json:"check"`
}

type DeviceOwner struct {
	Email string `json:"email"`
}

type Device struct {
	Id              int              `json:"id"`
	Name            string           `json:"name"`
	OwnedBy         string           `json:"owned_by"`
	Platform        string           `json:"platform"`
	LastSeenAt      *time.Time       `json:"last_seen_at"`
	FailureCount    int              `json:"failure_count"`
	PrimaryUserName string           `json:"primary_user_name"`
	Serial          string           `json:"serial"`
	AssignedOwner   DeviceOwner      `json:"assigned_owner"`
	Failures        []*DeviceFailure `json:"failures"`
}

type Check struct {
	Tags []string `json:"tags"`
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

const (
	DurationNotice   = time.Hour * 24 * 7
	DurationWarning  = time.Hour * 24 * 2
	DurationDanger   = time.Hour
	DurationCritical = 0
	DurationUnknown  = time.Hour * 24 * 30
)
