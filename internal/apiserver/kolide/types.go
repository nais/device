package kolide

import (
	"encoding/json"
	"time"
)

type Reference struct {
	Identifier string `json:"identifier"`
}

type Issue struct {
	ID              string         `json:"id"`
	Value           map[string]any `json:"value"`
	Title           string         `json:"title"`
	DetectedAt      *time.Time     `json:"detected_at"`
	ResolvedAt      *time.Time     `json:"resolved_at"`
	LastRecheckedAt *time.Time     `json:"last_rechecked_at"`
	Exempted        bool           `json:"exempted"`
	DeviceRef       Reference      `json:"device_information"`
	CheckRef        Reference      `json:"check_information"`
}

type Person struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type Device struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Platform            string     `json:"device_type"`
	Serial              string     `json:"serial"`
	LastAuthenticatedAt *time.Time `json:"last_authenticated_at"`

	OwnerRef Reference `json:"registered_owner_info"`
	Owner    Person
}

type CheckTag struct {
	Name string `json:"name"`
}

type Check struct {
	ID          string     `json:"id"`
	Tags        []CheckTag `json:"check_tags"`
	IssueTitle  string     `json:"issue_title"`
	Description string     `json:"description"`
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
