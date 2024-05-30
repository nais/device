package kolide

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
)

type Client interface {
	RefreshCache(ctx context.Context) error
	GetDevice(ctx context.Context, email, platform, serial string) (Device, error)
}

type client struct {
	baseUrl string
	client  *http.Client

	checks  *Cache[uint64, Check]
	devices *Cache[string, Device]

	log logrus.FieldLogger
}

func New(token string) Client {
	return &client{
		baseUrl: "https://k2.kolide.com/api/v0",
		client: &http.Client{
			Transport: NewTransport(token),
		},
		checks:  &Cache[uint64, Check]{},
		devices: &Cache[string, Device]{},
	}
}

func deviceKey(email, platform, serial string) string {
	return strings.ToLower(fmt.Sprintf("%v-%v-%v", email, platform, serial))
}

func (kc *client) RefreshCache(ctx context.Context) error {
	checks, err := kc.getChecks(ctx)
	if err != nil {
		return fmt.Errorf("getting checks: %w", err)
	}

	checksCache := make(map[uint64]Check, len(checks))
	for _, check := range checks {
		checksCache[check.ID] = check
	}

	kc.checks.Replace(checksCache)

	devices, err := kc.getDevices(ctx)
	if err != nil {
		return fmt.Errorf("getting devices: %w", err)
	}

	devicesCache := make(map[string]Device, len(devices))
	for _, device := range devices {
		devicesCache[deviceKey(device.AssignedOwner.Email, device.Platform, device.Serial)] = device
	}

	kc.devices.Replace(devicesCache)

	return nil
}

func (kc *client) GetDevice(ctx context.Context, email, platform, serial string) (Device, error) {
	key := deviceKey(email, platform, serial)
	device, ok := kc.devices.Get(key)
	if !ok {
		return Device{}, fmt.Errorf("device with key %v not found in cache", key)
	}

	failures, err := kc.getDeviceFailures(ctx, device.ID)
	if err != nil {
		return Device{}, fmt.Errorf("getting device failures: %w", err)
	}

	device.Failures = failures
	return device, nil
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
	nextUrl.Query().Set("per_page", "100")

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

		devices = append(devices, device)
	}

	return devices, nil
}

func (kc *client) getDeviceFailures(ctx context.Context, deviceID uint64) ([]DeviceFailure, error) {
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

		failure.Check, err = kc.getCheck(ctx, failure.Check.ID)
		if err != nil {
			return nil, fmt.Errorf("getting check: %w", err)
		}
		failures = append(failures, failure)
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
