package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nais/device/internal/apiserver/sqlc"
)

func (db *database) Approve(ctx context.Context, userID string) error {
	return db.queries.Approve(ctx, userID)
}

func (db *database) GetApprovals(ctx context.Context) (map[string]struct{}, error) {
	rows, err := db.queries.GetApprovals(ctx)
	if err != nil {
		return nil, fmt.Errorf("get approvals: %w", err)
	}

	res := make(map[string]struct{})
	for _, r := range rows {
		res[r.UserID] = struct{}{}
	}

	return res, nil
}

func (db *database) GetApproval(ctx context.Context, userID string) (*sqlc.Approval, error) {
	row, err := db.queries.GetApproval(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	return row, err
}
