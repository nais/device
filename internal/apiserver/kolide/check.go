package kolide

import (
	"strings"
	"time"

	"github.com/nais/kolide-event-handler/pkg/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (check Check) Severity() Severity {
	var severity, highest Severity = -1, -1

	for _, tag := range check.Tags {
		switch strings.ToLower(tag) {
		case "info":
			severity = SeverityInfo
		case "notice":
			severity = SeverityNotice
		case "warning":
			severity = SeverityWarning
		case "danger":
			severity = SeverityDanger
		case "critical":
			severity = SeverityCritical
		}

		if severity > highest {
			highest = severity
		}
	}

	if highest == -1 {
		log.Warnf("Check missing a severity tag: %+v", check)
		highest = SeverityWarning
	}

	return highest
}

func (severity Severity) GraceTime() time.Duration {
	switch severity {
	case SeverityNotice:
		return DurationNotice
	case SeverityWarning:
		return DurationWarning
	case SeverityDanger:
		return DurationDanger
	case SeverityCritical:
		return DurationCritical
	default:
		return DurationUnknown
	}
}

func (failure *DeviceFailure) Health() pb.Health {
	if failure == nil || failure.Ignored || failure.ResolvedAt != nil {
		return pb.Health_Healthy
	}

	if failure.Check.ID == 0 {
		log.Errorf("BUG: malformed failure from Kolide API: failure=%d; checkID=%d", failure.ID, failure.CheckID)
		return pb.Health_Healthy
	}

	// Ignore INFO checks
	severity := failure.Check.Severity()
	if severity == SeverityInfo {
		return pb.Health_Healthy
	}

	graceTime := severity.GraceTime()
	if graceTime == DurationUnknown {
		log.Errorf("DurationUnknown grace time for check %d, with tags: %+v", failure.CheckID, failure.Check.Tags)
	}

	// Deny by default if check time is unknown; might have been a long time ago
	if failure.Timestamp == nil {
		return pb.Health_Unhealthy
	}

	deadline := failure.Timestamp.Add(graceTime)
	if time.Now().After(deadline) {
		return pb.Health_Unhealthy
	}

	return pb.Health_Healthy
}

const MaxTimeSinceKolideLastSeen = 25 * time.Hour

// If one check fails, the device is unhealthy.
func (device *Device) Health() (pb.Health, string) {
	// Allow only registered devices
	if len(device.AssignedOwner.Email) == 0 {
		return pb.Health_Unhealthy, "Kolide does not know who owns this device"
	}

	// Devices must phone home regularly
	lastSeen := time.Time{}
	if device.LastSeenAt != nil {
		lastSeen = *device.LastSeenAt
	}
	deadline := lastSeen.Add(MaxTimeSinceKolideLastSeen)
	if time.Now().After(deadline) {
		msg := "Kolide's information about this device is out of date. Make sure the Kolide Launcher is running."
		return pb.Health_Unhealthy, msg
	}

	// Any failure means device failure
	for _, failure := range device.Failures {
		if failure.Health() == pb.Health_Unhealthy {
			return pb.Health_Unhealthy, failure.Title
		}
	}

	return pb.Health_Healthy, ""
}

func (device *Device) Event() *pb.DeviceEvent {
	health, msg := device.Health()
	return &pb.DeviceEvent{
		Timestamp: timestamppb.Now(),
		Serial:    device.Serial,
		Platform:  device.Platform,
		State:     health,
		Message:   msg,
	}
}
