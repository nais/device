package jita

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nais/device/pkg/ioconvenience"
)

type PrivilegedUser struct {
	UserId string `json:"user_id"`
}

func (j *client) GetPrivilegedUsersForGateway(gateway string) []PrivilegedUser {
	j.lock.RLock()
	defer j.lock.RUnlock()

	return j.PrivilegedUsers[gateway]
}

func (j *client) UpdatePrivilegedUsers() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s", j.URL, "gatewaysAccess"), nil)
	if err != nil {
		return fmt.Errorf("make jita request: %w", err)
	}
	resp, err := j.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("getting all privileged users: %w", err)
	}

	defer ioconvenience.CloseWithLog(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("not ok when calling jita: %v", resp.StatusCode)
	}

	update := map[string][]PrivilegedUser{}
	if err := json.NewDecoder(resp.Body).Decode(&update); err != nil {
		return fmt.Errorf("decoding all privileged users: %w", err)
	}

	j.lock.Lock()
	j.PrivilegedUsers = update
	j.lock.Unlock()

	return nil
}
