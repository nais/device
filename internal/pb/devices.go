package pb

import (
	"fmt"
	"slices"
	"time"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// Satisfy WireGuard interface.
// This value is written to the config file as a comment, so we put in the serial of the device in order to identify it.
func (x *Device) GetName() string {
	return x.GetSerial()
}

// Satisfy WireGuard interface.
// This field contains the private IP addresses of a device.
func (x *Device) GetAllowedIPs() []string {
	ips := []string{x.GetIpv4() + "/32"}
	if x.GetIpv6() != "" {
		ips = append(ips, x.GetIpv6()+"/128")
	}
	return ips
}

// Satisfy WireGuard interface.
// Endpoints are not used when configuring gateway and api server; connections are initiated from the client.
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

const lastSeenGracePeriod = time.Hour

func (x *Device) MaybeLstSeenIssue() *DeviceIssue {
	if x == nil {
		return nil
	}

	if x.LastSeen == nil {
		return lastSeenIssue("This device has never been seen by Kolide. Enroll device by asking @Kolide for a new installer on Slack. `/msg @Kolide installers`", x.LastUpdated)
	}

	lastSeenAfter := time.Now().Add(-lastSeenGracePeriod)
	if x.LastSeen.AsTime().After(lastSeenAfter) {
		return nil
	}

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
	return lastSeenIssue(msg, x.LastSeen)
}

func lastSeenIssue(msg string, lastUpdated *timestamppb.Timestamp) *DeviceIssue {
	return &DeviceIssue{
		Title:         "Device has not been seen recently",
		Message:       msg,
		Severity:      Severity_Critical,
		DetectedAt:    lastUpdated,
		LastUpdated:   lastUpdated,
		ResolveBefore: timestamppb.New(time.Now().Add(-lastSeenGracePeriod)),
	}
}
