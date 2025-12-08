package kolide

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/nais/device/internal/ioconvenience"
	"github.com/sirupsen/logrus"
)

type Client interface {
	GetIssues(ctx context.Context) ([]*DeviceFailure, error)
	GetDeviceIssues(ctx context.Context, deviceID string) ([]*DeviceFailure, error)
	GetChecks(ctx context.Context) ([]*Check, error)
	GetDevices(ctx context.Context) ([]*Device, error)
}

type client struct {
	baseUrl string
	client  *http.Client

	log logrus.FieldLogger
}

type ClientOption func(*client)

func WithBaseUrl(baseUrl string) ClientOption {
	return func(c *client) {
		c.baseUrl = baseUrl
	}
}

func New(token string, log logrus.FieldLogger, opts ...ClientOption) *client {
	c := &client{
		baseUrl: "https://k2.kolide.com/api/v0",
		client: &http.Client{
			Transport: NewTransport(token),
		},
		log: log,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

var _ Client = &client{}

func convertPlatform(platform string) string {
	switch strings.ToLower(platform) {
	case "darwin":
		return "darwin"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

func (kc *client) GetIssues(ctx context.Context) ([]*DeviceFailure, error) {
	resp, err := kc.getPaginated(ctx, kc.baseUrl+"/failures/open")
	if err != nil {
		return nil, fmt.Errorf("getting open failures: %w", err)
	}

	issues := make([]*DeviceFailure, len(resp))
	for i, rawFailure := range resp {
		failure := DeviceFailure{}
		err := json.Unmarshal(rawFailure, &failure)
		if err != nil {
			return nil, fmt.Errorf("unmarshal failure: %w", err)
		}
		issues[i] = &failure
	}

	return issues, nil
}

func (kc *client) GetChecks(ctx context.Context) ([]*Check, error) {
	rawChecks, err := kc.getPaginated(ctx, kc.baseUrl+"/checks")
	if err != nil {
		return nil, fmt.Errorf("getting checks: %w", err)
	}

	checks := make([]*Check, len(rawChecks))
	for i, rawCheck := range rawChecks {
		check := &Check{}
		err := json.Unmarshal(rawCheck, check)
		if err != nil {
			return nil, err
		}
		checks[i] = check
	}
	return checks, nil
}

func (kc *client) getPaginated(ctx context.Context, initialUrl string) ([]json.RawMessage, error) {
	var data []json.RawMessage
	nextUrl, err := url.Parse(initialUrl)
	if err != nil {
		return nil, err
	}

	q := nextUrl.Query()
	q.Set("per_page", "100")
	nextUrl.RawQuery = q.Encode()

	for {
		err := func() error {
			response, err := kc.get(ctx, nextUrl.String())
			if err != nil {
				return fmt.Errorf("getting paginated response: %w", err)
			}

			defer ioconvenience.CloseWithLog(kc.log, response.Body)

			paginatedResponse := &PaginatedResponse{}
			err = json.NewDecoder(response.Body).Decode(paginatedResponse)
			if err != nil {
				return fmt.Errorf("decoding paginated response: %w", err)
			}

			data = append(data, paginatedResponse.Data...)

			values := nextUrl.Query()
			values.Set("cursor", paginatedResponse.Pagination.NextCursor)
			nextUrl.RawQuery = values.Encode()
			return nil
		}()
		if nextUrl.Query().Get("cursor") == "" || err != nil {
			return data, err
		}
	}
}

func (kc *client) GetDevices(ctx context.Context) ([]*Device, error) {
	kc.log.Debug("getting all devices...")
	url := kc.baseUrl + "/devices"
	rawDevices, err := kc.getPaginated(ctx, url)
	if err != nil {
		return nil, err
	}

	devices := make([]*Device, len(rawDevices))
	for i, rawDevice := range rawDevices {
		device := &Device{}
		err := json.Unmarshal(rawDevice, device)
		if err != nil {
			return nil, fmt.Errorf("unmarshal device: %w", err)
		}

		device.Platform = convertPlatform(device.Platform)
		devices[i] = device
	}

	return devices, nil
}

func (kc *client) GetDeviceIssues(ctx context.Context, deviceID string) ([]*DeviceFailure, error) {
	url := fmt.Sprintf(kc.baseUrl+"/devices/%v/failures", deviceID)
	rawFailures, err := kc.getPaginated(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting paginated device failures: %v", err)
	}

	failures := make([]*DeviceFailure, len(rawFailures))
	for i, rawFailure := range rawFailures {
		err := json.Unmarshal(rawFailure, failures[i])
		if err != nil {
			return nil, fmt.Errorf("unmarshal failure: %w", err)
		}
	}
	return failures, nil
}

func (kc *client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	return kc.client.Do(req)
}
