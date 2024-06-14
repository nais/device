package kolide_test

import (
	"strings"
	"testing"
	"time"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/nais/device/internal/pb"
	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	tagTests := []struct {
		tags     []string
		severity pb.Severity
		duration time.Duration
	}{
		{[]string{}, pb.Severity_Warning, kolide.DurationWarning},
		{[]string{"foo", "bar"}, pb.Severity_Warning, kolide.DurationWarning},
		{[]string{"foo", "notice"}, pb.Severity_Notice, kolide.DurationNotice},
		{[]string{"warning", "notice", "danger"}, pb.Severity_Danger, kolide.DurationDanger},
		{[]string{"notice"}, pb.Severity_Notice, kolide.DurationNotice},
		{[]string{"warning"}, pb.Severity_Warning, kolide.DurationWarning},
		{[]string{"danger"}, pb.Severity_Danger, kolide.DurationDanger},
		{[]string{"critical"}, pb.Severity_Critical, kolide.DurationCritical},
	}

	for _, tt := range tagTests {
		t.Run(strings.Join(tt.tags, ", "), func(t *testing.T) {
			check := kolide.Check{
				Tags: tt.tags,
			}

			severity := check.Severity()

			assert.Equal(t, tt.severity, severity)
			assert.Equal(t, tt.duration, kolide.GraceTime(severity))
		})
	}
}
