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
		log.Errorf("Unknown severity: %v", severity)
		return DurationUnknown
	}
}

func AfterGracePeriod(failure DeviceFailure) bool {
	severity := GetSeverity(*failure.Check)
	graceTime := GetGraceTime(severity)

	if failure.Timestamp == nil {
		return true
	}

	if time.Now().After(failure.Timestamp.Add(graceTime)) {
		return true
	} else {
		return false
	}
}
