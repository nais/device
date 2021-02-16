package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nais/device/pkg/pb"
)

type UnauthorizedError struct{}

func (u *UnauthorizedError) Error() string {
	return "unauthorized"
}

type UnhealthyError struct{}

func (e *UnhealthyError) Error() string {
	return "device is in unhealthy state"
}

func GetDeviceConfig(sessionKey, apiServerURL string, ctx context.Context) ([]*pb.Gateway, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	deviceConfigAPI := fmt.Sprintf("%s/deviceconfig", apiServerURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, deviceConfigAPI, nil)
	if err != nil {
		return nil, fmt.Errorf("creating get request: %w", err)
	}
	req.Header.Add("x-naisdevice-session-key", sessionKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting device config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized access from apiserver: %w", &UnauthorizedError{})
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("http response %v: %w", http.StatusText(resp.StatusCode), &UnhealthyError{})
	}

	var gateways []*pb.Gateway
	if err := json.NewDecoder(resp.Body).Decode(&gateways); err != nil {
		return nil, fmt.Errorf("unmarshalling response body into gateways: %w", err)
	}

	return gateways, nil
}
