package kolide

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client interface {
	RefreshCache(ctx context.Context) error
	FillKolideData(ctx context.Context, devices []*pb.Device) error
	DumpChecks() ([]byte, error)
	GetDeviceFailures(ctx context.Context, deviceID string) ([]*pb.DeviceIssue, error)
}

type client struct {
	baseUrl string
	client  *http.Client

	checks *Cache[uint64, Check]

	log logrus.FieldLogger
}

type ClientOption func(*client)

func WithBaseUrl(baseUrl string) ClientOption {
	return func(c *client) {
		c.baseUrl = baseUrl
	}
}

func New(token string, db database.Database, log logrus.FieldLogger, opts ...ClientOption) Client {
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
			kc.log.WithError(err).Error("getting checks")
		}
	}

	checksCache := make(map[uint64]Check, len(checks))
	for _, check := range checks {
		checksCache[check.ID] = check
	}

	kc.checks.Replace(checksCache)

	return nil
}

func (kc *client) getAllDeviceIssues(ctx context.Context) (map[string][]*pb.DeviceIssue, error) {
	resp, err := kc.getPaginated(ctx, kc.baseUrl+"/failures/open")
	if err != nil {
		return nil, fmt.Errorf("getting open failures: %w", err)
	}

	issues := make(map[string][]*pb.DeviceIssue)
	for _, rawFailure := range resp {
		failure := DeviceFailureWithDevice{}
		err := json.Unmarshal(rawFailure, &failure)
		if err != nil {
			return nil, fmt.Errorf("unmarshal failure: %w", err)
		}

		failure.Check, err = kc.getCheck(ctx, failure.CheckID)
		if err != nil {
			return nil, fmt.Errorf("getting check: %w", err)
		}

		if !failure.Relevant() {
			continue
		}

		externalID := fmt.Sprint(failure.Device.ID)
		issues[externalID] = append(issues[externalID], failure.AsDeviceIssue())
	}

	return issues, nil
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
	kc.log.WithField("id", checkID).Info("cache miss for check")

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

func (kc *client) FillKolideData(ctx context.Context, devices []*pb.Device) error {
	kolideDevices, err := kc.getDevices(ctx)
	if err != nil {
		return fmt.Errorf("getting devices: %w", err)
	}

	issuesByExternalID, err := kc.getAllDeviceIssues(ctx)
	if err != nil {
		return fmt.Errorf("getting device issues: %w", err)
	}

	type key struct {
		serial   string
		platform string
	}

	kolideDeviceByKey := make(map[key]*Device)
	for _, kolideDevice := range kolideDevices {
		kolideDeviceByKey[key{kolideDevice.Serial, kolideDevice.Platform}] = &kolideDevice
	}

	for _, device := range devices {
		kolideDevice, ok := kolideDeviceByKey[key{device.Serial, device.Platform}]
		if !ok {
			continue
		}

		device.ExternalID = fmt.Sprint(kolideDevice.ID)
		device.Issues = issuesByExternalID[device.ExternalID]
		if kolideDevice.LastSeenAt != nil {
			device.LastSeen = timestamppb.New(*kolideDevice.LastSeenAt)
		}
	}

	return nil
}

func (kc *client) getDevices(ctx context.Context) ([]Device, error) {
	kc.log.Debug("getting all devices...")
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

	var issues []*pb.DeviceIssue
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

		if !failure.Relevant() {
			continue
		}

		issues = append(issues, failure.AsDeviceIssue())
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

func (kc client) DumpChecks() ([]byte, error) {
	return json.Marshal(kc.checks)
}
