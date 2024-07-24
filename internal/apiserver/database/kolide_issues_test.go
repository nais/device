package database

import (
	"strings"
	"testing"
	"time"

	"github.com/nais/device/internal/pb"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	tagTests := []struct {
		tags     []string
		severity pb.Severity
		duration time.Duration
	}{
		{[]string{}, pb.Severity_Warning, DurationWarning},
		{[]string{"foo", "bar"}, pb.Severity_Warning, DurationWarning},
		{[]string{"foo", "notice"}, pb.Severity_Notice, DurationNotice},
		{[]string{"warning", "notice", "danger"}, pb.Severity_Danger, DurationDanger},
		{[]string{"notice"}, pb.Severity_Notice, DurationNotice},
		{[]string{"warning"}, pb.Severity_Warning, DurationWarning},
		{[]string{"danger"}, pb.Severity_Danger, DurationDanger},
		{[]string{"critical"}, pb.Severity_Critical, DurationCritical},
	}

	for _, tt := range tagTests {
		t.Run(strings.Join(tt.tags, ", "), func(t *testing.T) {
			severity := kolideCheckSeverity(tt.tags, logrus.New())

			assert.Equal(t, tt.severity, severity)
			assert.Equal(t, tt.duration, graceTime(severity))
		})
	}
}
