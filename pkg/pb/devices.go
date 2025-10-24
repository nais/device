package pb

import (
	"fmt"
	"slices"
	"time"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// GetName is written to the config file as a comment, so we put in the serial of the device in order to identify it.
func (x *Device) GetName() string {
	return x.GetSerial()
}

// GetAllowedIPs returns the private IP addresses of a device.
func (x *Device) GetAllowedIPs() []string {
	ips := []string{x.GetIpv4() + "/32"}
	if x.GetIpv6() != "" {
		ips = append(ips, x.GetIpv6()+"/128")
	}
	return ips
}

// GetEndpoint does not return anything for devices, as they have no known endpoints.
func (x *Device) GetEndpoint() string {
	return ""
}

func AfterGracePeriod(d *DeviceIssue) bool {
	return time.Now().After(d.GetResolveBefore().AsTime())
}

func (x *Device) Healthy() bool {
	if x == nil {
		return false
	}

	return !slices.ContainsFunc(x.GetIssues(), AfterGracePeriod)
}

const (
	lastSeenGracePeriod = time.Hour
	lastSeenIssueTitle  = "Device has not been seen recently"
)

func (x *Device) UpdateLastSeenIssues() {
	// no device, lol
	if x == nil {
		return
	}

	makeLastSeenIssue := func(msg string, lastUpdated *timestamppb.Timestamp) *DeviceIssue {
		return &DeviceIssue{
			Title:         lastSeenIssueTitle,
			Message:       msg,
			Severity:      Severity_Critical,
			DetectedAt:    lastUpdated,
			LastUpdated:   lastUpdated,
			ResolveBefore: timestamppb.New(time.Now().Add(-lastSeenGracePeriod)),
		}
	}

	identifyLastSeenIssues := func(d *DeviceIssue) bool { return d.GetTitle() == lastSeenIssueTitle }

	// never seen
	if x.LastSeen == nil {
		if !slices.ContainsFunc(x.Issues, identifyLastSeenIssues) {
			x.Issues = append(x.Issues, makeLastSeenIssue("This device has never been seen by Kolide. Enroll device by asking @Kolide for a new installer on Slack. `/msg @Kolide installers`", x.LastUpdated))
		}
		return
	}

	// seen recently
	if x.LastSeen.AsTime().After(time.Now().Add(-lastSeenGracePeriod)) {
		x.Issues = slices.DeleteFunc(x.Issues, identifyLastSeenIssues)
		return
	}

	// if we end up here, this device has not been seen recently

	// best effort to convert time to Oslo timezone
	lastSeen := x.LastSeen.AsTime()
	location, err := time.LoadLocation("Europe/Oslo")
	if err == nil {
		lastSeen = lastSeen.In(location)
	}

	msg := fmt.Sprintf(`This device has not been seen by Kolide since %v.
This is a problem because we have no idea what state the device is in.
To fix this make sure the Kolide launcher is running.
If it's not and you don't know why - re-install the launcher by asking @Kolide for a new installer on Slack.`, lastSeen.Format(time.RFC3339))
	x.Issues = append(x.Issues, makeLastSeenIssue(msg, x.LastUpdated))
}
