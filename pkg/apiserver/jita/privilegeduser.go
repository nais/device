package jita

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	resp, err := j.HTTPClient.Get(fmt.Sprintf("%s/%s", j.URL, "gatewaysAccess"))
	if err != nil {
		return fmt.Errorf("getting all privileged users: %w", err)
	}

	defer ioconvenience.CloseWithLog(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("not ok when calling jita: %v", resp.StatusCode)
	}

	j.lock.Lock()
	defer j.lock.Unlock()

	update := map[string][]PrivilegedUser{}
	if err := json.NewDecoder(resp.Body).Decode(&update); err != nil {
		return fmt.Errorf("decoding all privileged users: %w", err)
	}
	j.PrivilegedUsers = update

	return nil
}
