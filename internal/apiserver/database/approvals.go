package database

import (
	"context"
	"fmt"
)

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
