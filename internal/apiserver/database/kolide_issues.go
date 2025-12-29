package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/nais/device/internal/apiserver/sqlc"
	"github.com/nais/device/internal/formats"
	"github.com/nais/device/pkg/pb"
	"github.com/sirupsen/logrus"
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
		case strings.EqualFold(tag, pb.Severity_Attention.String()):
			severity = pb.Severity_Attention
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

func (db *database) UpdateKolideChecks(ctx context.Context, checks []*kolide.Check) error {
	for _, check := range checks {
		checkID, err := strconv.ParseInt(check.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("parse check ID: %w", err)
		}
		tagNames := make([]string, len(check.Tags))
		for i, tag := range check.Tags {
			tagNames[i] = tag.Name
		}
		if err := db.queries.SetKolideCheck(ctx, sqlc.SetKolideCheckParams{
			ID:          checkID,
			Tags:        strings.Join(tagNames, ","),
			DisplayName: check.IssueTitle,
			Description: check.Description,
		}); err != nil {
			return fmt.Errorf("upsert Kolide check: %w", err)
		}
	}
	return nil
}

func (db *database) UpdateKolideIssues(ctx context.Context, issues []*kolide.Issue) error {
	return db.queries.Transaction(ctx, func(ctx context.Context, qtx *sqlc.Queries) error {
		if err := qtx.TruncateKolideIssues(ctx); err != nil {
			return fmt.Errorf("truncate Kolide issues: %w", err)
		}
		for _, issue := range issues {
			issueID, err := strconv.ParseInt(issue.ID, 10, 64)
			if err != nil {
				return fmt.Errorf("parse issue ID: %w", err)
			}
			checkID, err := strconv.ParseInt(issue.CheckRef.Identifier, 10, 64)
			if err != nil {
				return fmt.Errorf("parse check ID: %w", err)
			}
			resolvedAt := sql.NullString{}
			if issue.ResolvedAt != nil {
				resolvedAt.String = issue.ResolvedAt.Format(formats.TimeFormat)
				resolvedAt.Valid = true
			}
			detectedAt := ""
			if issue.DetectedAt != nil {
				detectedAt = issue.DetectedAt.Format(formats.TimeFormat)
			}
			lastUpdated := ""
			if issue.LastRecheckedAt != nil {
				lastUpdated = issue.LastRecheckedAt.Format(formats.TimeFormat)
			}
			if err := qtx.SetKolideIssue(ctx, sqlc.SetKolideIssueParams{
				ID:          issueID,
				DeviceID:    issue.DeviceRef.Identifier,
				CheckID:     checkID,
				Title:       issue.Title,
				DetectedAt:  detectedAt,
				ResolvedAt:  resolvedAt,
				LastUpdated: lastUpdated,
				Ignored:     issue.Exempted,
			}); err != nil {
				return fmt.Errorf("upsert Kolide issue: %w", err)
			}
		}
		return nil
	})
}

func (db *database) UpdateKolideIssuesForDevice(ctx context.Context, externalID string, issues []*kolide.Issue) error {
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
			issueID, err := strconv.ParseInt(issue.ID, 10, 64)
			if err != nil {
				return fmt.Errorf("parse issue ID: %w", err)
			}
			checkID, err := strconv.ParseInt(issue.CheckRef.Identifier, 10, 64)
			if err != nil {
				return fmt.Errorf("parse check ID: %w", err)
			}
			resolvedAt := sql.NullString{}
			if issue.ResolvedAt != nil {
				resolvedAt.String = issue.ResolvedAt.Format(formats.TimeFormat)
				resolvedAt.Valid = true
			}
			detectedAt := ""
			if issue.DetectedAt != nil {
				detectedAt = issue.DetectedAt.Format(formats.TimeFormat)
			}
			lastUpdated := ""
			if issue.LastRecheckedAt != nil {
				lastUpdated = issue.LastRecheckedAt.Format(formats.TimeFormat)
			}
			params := sqlc.SetKolideIssueParams{
				ID:          issueID,
				DeviceID:    issue.DeviceRef.Identifier,
				CheckID:     checkID,
				Title:       issue.Title,
				DetectedAt:  detectedAt,
				ResolvedAt:  resolvedAt,
				LastUpdated: lastUpdated,
				Ignored:     issue.Exempted,
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
