package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/nais/device/internal/apiserver/sqlc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (db *database) AcceptAcceptableUse(ctx context.Context, userID string) error {
	return db.queries.AcceptAcceptableUse(ctx, sqlc.AcceptAcceptableUseParams{
		UserID:     userID,
		AcceptedAt: timeToString(time.Now()),
	})
}

func (db *database) RejectAcceptableUse(ctx context.Context, userID string) error {
	return db.queries.RejectAcceptableUse(ctx, userID)
}

func (db *database) GetAcceptances(ctx context.Context) (map[string]struct{}, error) {
	rows, err := db.queries.GetAcceptances(ctx)
	if err != nil {
		return nil, fmt.Errorf("get acceptances: %w", err)
	}

	res := make(map[string]struct{})
	for _, r := range rows {
		res[r.UserID] = struct{}{}
	}

	return res, nil
}

func (db *database) GetAcceptedAt(ctx context.Context, userID string) (*timestamppb.Timestamp, error) {
	row, err := db.queries.GetAcceptance(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return timestamppb.New(stringToTime(row.AcceptedAt)), nil
}
