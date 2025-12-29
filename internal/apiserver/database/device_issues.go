package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nais/device/internal/apiserver/sqlc"
	"github.com/nais/device/pkg/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Devices are allowed to connect this long after a failed check is triggered.
const (
	DurationCritical  = 0
	DurationDanger    = time.Hour
	DurationWarning   = time.Hour * 24 * 2
	DurationAttention = time.Hour * 24 * 3
	DurationNotice    = time.Hour * 24 * 7
	DurationUnknown   = time.Hour * 24 * 30
)

func (db *database) getDeviceIssues(ctx context.Context, device *sqlc.Device) ([]*pb.DeviceIssue, error) {
	if !db.kolideEnabled {
		return nil, nil
	}

	if device.ExternalID.String == "" {
		return []*pb.DeviceIssue{{
			Title:         "No Kolide device ID found for device",
			Message:       "Make sure you've installed Kolide according to the documentation at https://doc.nais.io/operate/naisdevice/how-to/install-kolide",
			Severity:      pb.Severity_Critical,
			DetectedAt:    timestamppb.Now(),
			LastUpdated:   timestamppb.Now(),
			ResolveBefore: timestamppb.New(time.Time{}),
		}}, nil
	}

	issues, err := db.queries.GetKolideIssuesForDevice(ctx, device.ExternalID.String)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("getting kolide issues: %w", err)
	}

	deviceIssues := make([]*pb.DeviceIssue, 0)

	for _, issue := range issues {
		checkTags := strings.Split(issue.Tags, ",")
		checkSeverity := kolideCheckSeverity(checkTags, db.log)

		if checkSeverity == pb.Severity_Info || issue.ResolvedAt.String != "" || issue.Ignored {
			continue
		}

		issueDetectedAt := stringToTime(issue.DetectedAt)
		issueLastUpdated := stringToTime(issue.LastUpdated)
		issueResolveBefore := issueDetectedAt.Add(graceTime(checkSeverity))

		deviceIssues = append(deviceIssues, &pb.DeviceIssue{
			Title:         issue.Title,
			Message:       issue.Description,
			Severity:      checkSeverity,
			DetectedAt:    timestamppb.New(issueDetectedAt),
			LastUpdated:   timestamppb.New(issueLastUpdated),
			ResolveBefore: timestamppb.New(issueResolveBefore),
		})
	}

	return deviceIssues, nil
}

func graceTime(severity pb.Severity) time.Duration {
	switch severity {
	case pb.Severity_Notice:
		return DurationNotice
	case pb.Severity_Attention:
		return DurationAttention
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
