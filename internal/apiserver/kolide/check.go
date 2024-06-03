package kolide

import (
	"strings"
	"time"

	"github.com/nais/device/internal/pb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (check Check) Severity() pb.Severity {
	highest := pb.Severity(-1)

	for _, tag := range check.Tags {
		severity := pb.Severity(-1)

		switch {
		case strings.EqualFold(tag, pb.Severity_Info.String()):
			severity = pb.Severity_Info
		case strings.EqualFold(tag, pb.Severity_Notice.String()):
			severity = pb.Severity_Notice
		case strings.EqualFold(tag, pb.Severity_Warning.String()):
			severity = pb.Severity_Warning
		case strings.EqualFold(tag, pb.Severity_Danger.String()):
			severity = pb.Severity_Danger
		case strings.EqualFold(tag, pb.Severity_Critical.String()):
			severity = pb.Severity_Critical
		default:
			log.Warnf("kolide severity parser: failed to parse tag: %q", tag)
		}

		if severity > highest {
			highest = severity
		}
	}

	if highest == -1 {
		log.Warnf("Check missing a severity tag: %+v", check)
		highest = pb.Severity_Warning
	}

	return highest
}

func GraceTime(severity pb.Severity) time.Duration {
	switch severity {
	case pb.Severity_Notice:
		return DurationNotice
	case pb.Severity_Warning:
		return DurationWarning
	case pb.Severity_Danger:
		return DurationDanger
	case pb.Severity_Critical:
		return DurationCritical
	default:
		return DurationUnknown
	}
}

func (failure *DeviceFailure) Relevant() bool {
	if failure == nil || failure.Ignored || failure.ResolvedAt != nil {
		return false
	}

	return failure.Check.Severity() != pb.Severity_Info
}

const MaxTimeSinceKolideLastSeen = 25 * time.Hour

// If one check fails, the device is unhealthy.
func (device *Device) Issues() []*pb.DeviceIssue {
	// Allow only registered devices
	if device.AssignedOwner.Email == "" {
		return []*pb.DeviceIssue{
			{
				Title:         "Device is not registered with an owner",
				Severity:      pb.Severity_Critical,
				Message:       "This device is not registered with an owner. Please talk with the Kolide bot on Slack.",
				DetectedAt:    timestamppb.Now(),
				ResolveBefore: timestamppb.New(time.Time{}),
				LastUpdated:   timestamppb.Now(),
			},
		}
	}

	// Devices must phone home regularly
	lastSeen := time.Time{}
	if device.LastSeenAt != nil {
		lastSeen = *device.LastSeenAt
	}
	deadline := lastSeen.Add(MaxTimeSinceKolideLastSeen)
	if time.Now().After(deadline) {
		return []*pb.DeviceIssue{
			{
				Title:         "Kolide's information about this device is out of date",
				Severity:      pb.Severity_Critical,
				Message:       "Kolide's information about this device is out of date. Make sure the Kolide Launcher is running.",
				DetectedAt:    timestamppb.Now(),
				ResolveBefore: timestamppb.New(time.Time{}),
				LastUpdated:   timestamppb.Now(),
			},
		}
	}

	// Any failure means device failure
	openIssues := []*pb.DeviceIssue{}
	for _, failure := range device.Failures {
		if !failure.Relevant() {
			continue
		}

		graceTime := GraceTime(failure.Check.Severity())
		if graceTime == DurationUnknown {
			log.Errorf("DurationUnknown grace time for check %+v", failure.Check.DisplayName)
		}

		var failureTimestamp *timestamppb.Timestamp
		var deadline *timestamppb.Timestamp
		if failure.Timestamp != nil {
			failureTimestamp = timestamppb.New(*failure.Timestamp)
			deadline = timestamppb.New(failure.Timestamp.Add(graceTime))
		} else {
			log.Warnf("Timestamp missing for failure %+v", failure)
		}

		openIssues = append(openIssues, &pb.DeviceIssue{
			Title:         failure.Title,
			Severity:      failure.Check.Severity(),
			Message:       failure.Check.Description,
			DetectedAt:    failureTimestamp,
			ResolveBefore: deadline,
			LastUpdated:   timestamppb.New(failure.LastUpdated),
		})
	}

	return openIssues
}
