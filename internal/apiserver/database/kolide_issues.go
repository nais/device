package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/nais/device/internal/apiserver/sqlc"
	"github.com/nais/device/internal/formats"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	lastSeen := stringToTime(device.LastSeen.String)
	lastUpdated := stringToTime(device.LastUpdated.String)

	issues, err := db.queries.GetKolideIssuesForDevice(ctx, device.ExternalID.String)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("getting kolide issues: %w", err)
	}

	deviceIssues := make([]*pb.DeviceIssue, 0)
	if lastSeenIssue := generateLastSeenIssue(lastSeen, lastUpdated); lastSeenIssue != nil {
		deviceIssues = append(deviceIssues, lastSeenIssue)
	}

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

const (
	lastSeenGracePeriod = 48 * time.Hour
	lastSeenIssueTitle  = "Device has not been seen recently"
)

func generateLastSeenIssue(lastSeen, lastUpdated time.Time) *pb.DeviceIssue {
	template := func(msg string, lastUpdated time.Time) *pb.DeviceIssue {
		return &pb.DeviceIssue{
			Title:         lastSeenIssueTitle,
			Message:       msg,
			Severity:      pb.Severity_Critical,
			DetectedAt:    timestamppb.New(lastUpdated),
			LastUpdated:   timestamppb.New(lastUpdated),
			ResolveBefore: timestamppb.New(time.Now().Add(-lastSeenGracePeriod)),
		}
	}

	if lastUpdated.IsZero() || lastSeen.IsZero() {
		return template("This device has never been seen by Kolide. Enroll device by asking @Kolide for a new installer on Slack. `/msg @Kolide installers`", lastUpdated)
	}

	// seen recently
	if lastSeen.After(time.Now().Add(-lastSeenGracePeriod)) {
		return nil
	}

	// if we end up here, this device has not been seen recently

	// best effort to convert time to Oslo timezone
	lastSeenIn := lastSeen
	location, err := time.LoadLocation("Europe/Oslo")
	if err == nil {
		lastSeenIn = lastSeen.In(location)
	}

	msg := fmt.Sprintf(`This device has not been seen by Kolide since %v.
This is a problem because we have no idea what state the device is in.
To fix this make sure the Kolide launcher is running.
If it's not and you don't know why - re-install the launcher by asking @Kolide for a new installer on Slack.`, lastSeenIn.Format(time.RFC3339))
	return template(msg, lastUpdated)
}

// Devices are allowed to connect this long after a failed check is triggered.
const (
	DurationCritical = 0
	DurationDanger   = time.Hour
	DurationWarning  = time.Hour * 24 * 2
	DurationNotice   = time.Hour * 24 * 7
	DurationUnknown  = time.Hour * 24 * 30
)

func kolideCheckSeverity(tags []string, log logrus.FieldLogger) pb.Severity {
	highest := pb.Severity(-1)

	for _, tag := range tags {
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
		highest = pb.Severity_Warning
	}

	return highest
}

func graceTime(severity pb.Severity) time.Duration {
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

func (db *database) UpdateKolideChecks(ctx context.Context, checks []*kolide.Check) error {
	for _, check := range checks {
		if err := db.queries.SetKolideCheck(ctx, sqlc.SetKolideCheckParams{
			ID:          check.ID,
			Tags:        strings.Join(check.Tags, ","),
			DisplayName: check.DisplayName,
			Description: check.Description,
		}); err != nil {
			return fmt.Errorf("upsert Kolide check: %w", err)
		}
	}
	return nil
}

func (db *database) UpdateKolideIssues(ctx context.Context, issues []*kolide.DeviceFailure) error {
	return db.queries.Transaction(ctx, func(ctx context.Context, qtx *sqlc.Queries) error {
		if err := qtx.TruncateKolideIssues(ctx); err != nil {
			return fmt.Errorf("truncate Kolide issues: %w", err)
		}
		for _, issue := range issues {
			resolvedAt := sql.NullString{}
			if issue.ResolvedAt != nil {
				resolvedAt.String = issue.ResolvedAt.Format(formats.TimeFormat)
				resolvedAt.Valid = true
			}
			if err := qtx.SetKolideIssue(ctx, sqlc.SetKolideIssueParams{
				ID:          issue.ID,
				DeviceID:    fmt.Sprint(issue.Device.ID),
				CheckID:     issue.CheckID,
				Title:       issue.Title,
				DetectedAt:  issue.Timestamp.Format(formats.TimeFormat),
				ResolvedAt:  resolvedAt,
				LastUpdated: issue.LastUpdated.Format(formats.TimeFormat),
				Ignored:     issue.Ignored,
			}); err != nil {
				return fmt.Errorf("upsert Kolide issue: %w", err)
			}
		}
		return nil
	})
}

func (db *database) UpdateKolideIssuesForDevice(ctx context.Context, externalID string, issues []*kolide.DeviceFailure) error {
	if !db.kolideEnabled {
		return nil
	}

	if externalID == "" {
		return fmt.Errorf("updating kolide issues for device: external ID is empty")
	}

	err := db.queries.Transaction(ctx, func(ctx context.Context, queries *sqlc.Queries) error {
		if err := db.queries.DeleteKolideIssuesForDevice(ctx, externalID); err != nil {
			return err
		}

		for _, issue := range issues {
			resolvedAt := sql.NullString{}
			if issue.ResolvedAt != nil {
				resolvedAt.String = issue.ResolvedAt.Format(formats.TimeFormat)
				resolvedAt.Valid = true
			}
			params := sqlc.SetKolideIssueParams{
				ID:          issue.ID,
				DeviceID:    fmt.Sprint(issue.Device.ID),
				CheckID:     issue.CheckID,
				Title:       issue.Title,
				DetectedAt:  issue.Timestamp.Format(formats.TimeFormat),
				ResolvedAt:  resolvedAt,
				LastUpdated: issue.LastUpdated.Format(formats.TimeFormat),
				Ignored:     issue.Ignored,
			}
			if err := db.queries.SetKolideIssue(ctx, params); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("updating kolide issues for device: %v: %w", externalID, err)
	}

	return nil
}
