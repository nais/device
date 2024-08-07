package kolide

import (
	"encoding/json"
	"time"
)

type DeviceFailure struct {
	ID          int64                  `json:"id"`
	CheckID     int64                  `json:"check_id"`
	Value       map[string]interface{} `json:"value"`
	Title       string                 `json:"title"`
	Timestamp   *time.Time             `json:"timestamp"`
	ResolvedAt  *time.Time             `json:"resolved_at"`
	LastUpdated time.Time              `json:"-"`
	Ignored     bool                   `json:"ignored"`
	Check       Check                  `json:"check"`
	Device      Device                 `json:"device"`
}

type DeviceOwner struct {
	Email string `json:"email"`
}

type Device struct {
	ID              int64           `json:"id"`
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
	ID          int64    `json:"id"`
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
