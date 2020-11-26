package jita

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type PrivilegedUser struct {
	UserId string `json:"user_id"`
}

func (j *Jita) GetPrivilegedUsersForGateway(gateway string) ([]PrivilegedUser, error) {

	resp, err := j.HTTPClient.Get(fmt.Sprintf("%s/%s/%s", j.Url, "gatewayAccess", gateway))
	if err != nil {
		return nil, fmt.Errorf("getting privileged users: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not ok when calling jita: %v", resp.StatusCode)
	}
	var users []PrivilegedUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("decoding privileged users: %w", err)
	}
	return users, nil
}
