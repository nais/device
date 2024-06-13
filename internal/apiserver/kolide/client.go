package kolide

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
)

type Client interface {
	RefreshCache(ctx context.Context) error
	DumpChecks() ([]byte, error)
	GetDeviceFailures(ctx context.Context, deviceID string) ([]*pb.DeviceIssue, error)
}

type client struct {
	baseUrl string
	client  *http.Client

	checks *Cache[uint64, Check]

	log logrus.FieldLogger
	db  database.APIServer
}

type ClientOption func(*client)

func WithBaseUrl(baseUrl string) ClientOption {
	return func(c *client) {
		c.baseUrl = baseUrl
	}
}

func New(token string, log logrus.FieldLogger, opts ...ClientOption) Client {
	c := &client{
		baseUrl: "https://k2.kolide.com/api/v0",
		client: &http.Client{
			Transport: NewTransport(token),
		},
		checks: &Cache[uint64, Check]{},
		log:    log,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

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

func (kc *client) RefreshCache(ctx context.Context) error {
	checks, err := kc.getChecks(ctx)
	if err != nil {
		if kc.checks.Len() == 0 {
			return fmt.Errorf("getting checks: %w", err)
		} else {
			kc.log.Errorf("getting checks: %v", err)
		}
	}

	checksCache := make(map[uint64]Check, len(checks))
	for _, check := range checks {
		checksCache[check.ID] = check
	}

	kc.checks.Replace(checksCache)

	devices, err := kc.getDevices(ctx)
	if err != nil {
		kc.log.Errorf("getting devices: %v", err)
	} else {
		for _, device := range devices {
			_, err := kc.db.UpdateSingleDevice(ctx, fmt.Sprint(device.ID), device.Serial, device.Platform, device.LastSeenAt, nil)
			if err != nil {
				kc.log.Errorf("storing device: %v", err)
			}
		}
	}

	if err := kc.updateDeviceFailures(ctx); err != nil {
		kc.log.Errorf("updating device failures: %v", err)
	}

	return nil
}

func (kc *client) updateDeviceFailures(ctx context.Context) error {
	resp, err := kc.getPaginated(ctx, kc.baseUrl+"/failures/open")
	if err != nil {
		return fmt.Errorf("getting open failures: %w", err)
	}

	type deviceKey struct {
		deviceID   string
		platform   string
		serial     string
		lastSeenAt *time.Time
	}

	devices := make(map[deviceKey][]DeviceFailure)
	for _, rawFailure := range resp {
		failure := DeviceFailureWithDevice{}
		err := json.Unmarshal(rawFailure, &failure)
		if err != nil {
			return fmt.Errorf("unmarshal failure: %w", err)
		}

		failure.Check, err = kc.getCheck(ctx, failure.CheckID)
		if err != nil {
			return fmt.Errorf("getting check: %w", err)
		}

		key := deviceKey{
			deviceID:   fmt.Sprint(failure.Device.ID),
			platform:   convertPlatform(failure.Device.Platform),
			serial:     failure.Device.Serial,
			lastSeenAt: failure.Device.LastSeenAt,
		}
		devices[key] = append(devices[key], failure.DeviceFailure)
	}

	checkedDevices := []int64{}
	for device, failures := range devices {
		issues := convertFailuresToOpenDeviceIssues(failures)
		id, err := kc.db.UpdateSingleDevice(ctx, device.deviceID, device.serial, device.platform, device.lastSeenAt, issues)
		if err != nil {
			kc.log.Errorf("storing device issues: %v", err)
			continue
		}
		checkedDevices = append(checkedDevices, id)
	}

	if len(checkedDevices) > 0 {
		err := kc.db.ClearDeviceIssuesExceptFor(ctx, checkedDevices)
		if err != nil {
			return fmt.Errorf("clearing device issues: %w", err)
		}
	}

	return nil
}

func (kc *client) getChecks(ctx context.Context) ([]Check, error) {
	rawChecks, err := kc.getPaginated(ctx, kc.baseUrl+"/checks")
	if err != nil {
		return nil, fmt.Errorf("getting checks: %w", err)
	}

	checks := make([]Check, len(rawChecks))
	for i, rawCheck := range rawChecks {
		check := Check{}
		err := json.Unmarshal(rawCheck, &check)
		if err != nil {
			return nil, err
		}
		checks[i] = check
	}
	return checks, nil
}

func (kc *client) getCheck(ctx context.Context, checkID uint64) (Check, error) {
	check, ok := kc.checks.Get(checkID)
	if ok {
		return check, nil
	}
	kc.log.Infof("cache miss for check with id: %v", checkID)

	url := fmt.Sprintf(kc.baseUrl+"/checks/%v", checkID)
	response, err := kc.get(ctx, url)
	if err != nil {
		return Check{}, fmt.Errorf("getting check: %w", err)
	}

	defer response.Body.Close()

	check = Check{}
	err = json.NewDecoder(response.Body).Decode(&check)
	if err != nil {
		return Check{}, fmt.Errorf("decoding check: %w", err)
	}

	kc.checks.Set(check.ID, check)

	return check, nil
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

			defer response.Body.Close()

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

func (kc *client) getDevices(ctx context.Context) ([]Device, error) {
	kc.log.Debugf("Getting all devices...")
	url := kc.baseUrl + "/devices"
	rawDevices, err := kc.getPaginated(ctx, url)
	if err != nil {
		return nil, err
	}

	var devices []Device
	for _, rawDevice := range rawDevices {
		device := Device{}
		err := json.Unmarshal(rawDevice, &device)
		if err != nil {
			return nil, fmt.Errorf("unmarshal device: %w", err)
		}

		device.Platform = convertPlatform(device.Platform)
		devices = append(devices, device)
	}

	return devices, nil
}

func (kc *client) GetDeviceFailures(ctx context.Context, deviceID string) ([]*pb.DeviceIssue, error) {
	url := fmt.Sprintf(kc.baseUrl+"/devices/%v/failures", deviceID)
	rawFailures, err := kc.getPaginated(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("getting paginated device failures: %v", err)
	}

	var failures []DeviceFailure
	for _, rawFailure := range rawFailures {
		failure := DeviceFailure{}
		err := json.Unmarshal(rawFailure, &failure)
		if err != nil {
			return nil, fmt.Errorf("unmarshal failure: %w", err)
		}

		failure.Check, err = kc.getCheck(ctx, failure.CheckID)
		if err != nil {
			return nil, fmt.Errorf("getting check: %w", err)
		}

		failures = append(failures, failure)
	}

	return convertFailuresToOpenDeviceIssues(failures), nil
}

func (kc *client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	return kc.client.Do(req)
}

func (kc client) DumpChecks() ([]byte, error) {
	return json.Marshal(kc.checks)
}
