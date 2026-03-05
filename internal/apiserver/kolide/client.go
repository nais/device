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
	GetIssues(ctx context.Context) ([]*Issue, error)
	GetDeviceIssues(ctx context.Context, deviceID string) ([]*Issue, error)
	GetChecks(ctx context.Context) ([]*Check, error)
	GetDevice(ctx context.Context, deviceID string) (*Device, error)
	GetDevices(ctx context.Context) ([]*Device, error)
	GetPeople(ctx context.Context) (map[string]*Person, error)
}

type client struct {
	baseURL string
	client  *http.Client

	log logrus.FieldLogger
}

type ClientOption func(*client)

func WithBaseURL(baseURL string) ClientOption {
	return func(c *client) {
		c.baseURL = baseURL
	}
}

func New(token string, log logrus.FieldLogger, opts ...ClientOption) *client {
	c := &client{
		baseURL: "https://api.kolide.com",
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
	case "darwin", "mac":
		return "darwin"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

func (kc *client) GetIssues(ctx context.Context) ([]*Issue, error) {
	resp, err := kc.getPaginated(ctx, kc.baseURL+"/issues?&query=resolved_at%3Anull")
	if err != nil {
		return nil, fmt.Errorf("getting open issues: %w", err)
	}

	issues := make([]*Issue, len(resp))
	for i, rawIssue := range resp {
		issue := Issue{}
		err := json.Unmarshal(rawIssue, &issue)
		if err != nil {
			return nil, fmt.Errorf("unmarshal issue: %w", err)
		}
		issues[i] = &issue
	}

	return issues, nil
}

func (kc *client) GetChecks(ctx context.Context) ([]*Check, error) {
	rawChecks, err := kc.getPaginated(ctx, kc.baseURL+"/checks")
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

func (kc *client) getPaginated(ctx context.Context, initialURL string) ([]json.RawMessage, error) {
	var data []json.RawMessage
	nextURL, err := url.Parse(initialURL)
	if err != nil {
		return nil, err
	}

	q := nextURL.Query()
	q.Set("per_page", "100")
	nextURL.RawQuery = q.Encode()

	for {
		err := func() error {
			response, err := kc.get(ctx, nextURL.String())
			if err != nil {
				return fmt.Errorf("getting paginated response: %w", err)
			}

			defer ioconvenience.CloseWithLog(response.Body, kc.log)

			paginatedResponse := &PaginatedResponse{}
			err = json.NewDecoder(response.Body).Decode(paginatedResponse)
			if err != nil {
				return fmt.Errorf("decoding paginated response: %w", err)
			}

			data = append(data, paginatedResponse.Data...)

			values := nextURL.Query()
			values.Set("cursor", paginatedResponse.Pagination.NextCursor)
			nextURL.RawQuery = values.Encode()
			return nil
		}()
		if nextURL.Query().Get("cursor") == "" || err != nil {
			return data, err
		}
	}
}

func (kc *client) GetPeople(ctx context.Context) (map[string]*Person, error) {
	rawPeople, err := kc.getPaginated(ctx, kc.baseURL+"/people")
	if err != nil {
		return nil, fmt.Errorf("getting people: %w", err)
	}

	people := make(map[string]*Person, len(rawPeople))
	for _, rawPerson := range rawPeople {
		person := &Person{}
		err := json.Unmarshal(rawPerson, person)
		if err != nil {
			return nil, fmt.Errorf("unmarshal person: %w", err)
		}
		people[person.ID] = person
	}
	return people, nil
}

func (kc *client) GetDevices(ctx context.Context) ([]*Device, error) {
	kc.log.Debug("getting all devices...")
	url := kc.baseURL + "/devices"
	rawDevices, err := kc.getPaginated(ctx, url)
	if err != nil {
		return nil, err
	}

	people, err := kc.GetPeople(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting people for device owner resolution: %w", err)
	}

	devices := make([]*Device, len(rawDevices))
	for i, rawDevice := range rawDevices {
		device := &Device{}
		err := json.Unmarshal(rawDevice, device)
		if err != nil {
			return nil, fmt.Errorf("unmarshal device: %w", err)
		}

		device.Platform = convertPlatform(device.Platform)
		if person, ok := people[device.OwnerRef.Identifier]; ok {
			device.Owner = *person
		}
		devices[i] = device
	}

	return devices, nil
}

func (kc *client) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	resp, err := kc.get(ctx, fmt.Sprintf(kc.baseURL+"/devices/%v", deviceID))
	if err != nil {
		return nil, fmt.Errorf("getting device %v: %w", deviceID, err)
	}
	defer ioconvenience.CloseWithLog(resp.Body, kc.log)

	device := &Device{}
	if err := json.NewDecoder(resp.Body).Decode(device); err != nil {
		return nil, fmt.Errorf("unmarshal device %v: %w", deviceID, err)
	}

	device.Platform = convertPlatform(device.Platform)

	if device.OwnerRef.Identifier != "" {
		resp, err := kc.get(ctx, fmt.Sprintf(kc.baseURL+"/people/%v", device.OwnerRef.Identifier))
		if err != nil {
			return nil, fmt.Errorf("getting person %v for device %v: %w", device.OwnerRef.Identifier, deviceID, err)
		}
		defer ioconvenience.CloseWithLog(resp.Body, kc.log)

		person := &Person{}
		if err := json.NewDecoder(resp.Body).Decode(person); err != nil {
			return nil, fmt.Errorf("unmarshal person %v: %w", device.OwnerRef.Identifier, err)
		}
		device.Owner = *person
	}

	return device, nil
}

func (kc *client) GetDeviceIssues(ctx context.Context, deviceID string) ([]*Issue, error) {
	url := fmt.Sprintf(kc.baseURL+"/devices/%v/open_issues", deviceID)
	rawIssues, err := kc.getPaginated(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting paginated device issues: %v", err)
	}

	issues := make([]*Issue, len(rawIssues))
	for i, rawIssue := range rawIssues {
		issue := &Issue{}
		err := json.Unmarshal(rawIssue, issue)
		if err != nil {
			return nil, fmt.Errorf("unmarshal issue: %w", err)
		}
		issues[i] = issue
	}
	return issues, nil
}

func (kc *client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	return kc.client.Do(req)
}
