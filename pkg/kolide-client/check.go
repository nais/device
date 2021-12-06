package kolide_client

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func GetSeverity(check Check) Severity {
	var severity, mostSevereSeverity Severity = -1, -1

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

		if severity > mostSevereSeverity {
			mostSevereSeverity = severity
		}
	}

	if mostSevereSeverity == -1 {
		log.Warnf("Check missing a severity tag: %+v", check)
		mostSevereSeverity = SeverityWarning
	}

	return mostSevereSeverity
}

func GetGraceTime(severity Severity) time.Duration {
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

func AfterGracePeriod(failure DeviceFailure) bool {
	if failure.Check == nil {
		log.Errorf("BUG: This should not happen, checking grace period for failure %d - Check is nil! (checkID is: %d)", failure.Id, failure.CheckId)
		return false
	}

	severity := GetSeverity(*failure.Check)
	if severity == SeverityInfo {
		return false
	}

	graceTime := GetGraceTime(severity)
	if graceTime == DurationUnknown {
		log.Errorf("DurationUnknown grace time for check %d, with tags: %+v", failure.CheckId, failure.Check.Tags)
	}

	if failure.Timestamp == nil {
		return true
	}

	if time.Now().After(failure.Timestamp.Add(graceTime)) {
		return true
	} else {
		return false
	}
}
