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
			log.WithField("tag", tag).Warn("Kolide severity parser: failed to parse tag")
		}

		if severity > highest {
			highest = severity
		}
	}

	if highest == -1 {
		log.WithField("check", check).Warn("Check missing a severity tag")
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

func (f DeviceFailure) AsDeviceIssue() *pb.DeviceIssue {
	graceTime := GraceTime(f.Check.Severity())
	if graceTime == DurationUnknown {
		log.WithField("check_name", f.Check.DisplayName).Error("DurationUnknown grace time for check")
	}

	var failureTimestamp *timestamppb.Timestamp
	var deadline *timestamppb.Timestamp
	if f.Timestamp != nil {
		failureTimestamp = timestamppb.New(*f.Timestamp)
		deadline = timestamppb.New(f.Timestamp.Add(graceTime))
	} else {
		log.WithField("failure", f).Warn("timestamp missing for failure")
	}

	return &pb.DeviceIssue{
		Title:         f.Title,
		Severity:      f.Check.Severity(),
		Message:       f.Check.Description,
		DetectedAt:    failureTimestamp,
		ResolveBefore: deadline,
		LastUpdated:   timestamppb.New(f.LastUpdated),
	}
}
