package database

import (
	"context"
	"database/sql"

	"github.com/nais/device/internal/apiserver/sqlc"
)

type Queries struct {
	*sqlc.Queries
	db *sql.DB
}

type Querier interface {
	sqlc.Querier
	Transaction(ctx context.Context, callback func(ctx context.Context, queries *sqlc.Queries) error) error
}

func NewQuerier(db *sql.DB) *Queries {
	return &Queries{
		Queries: sqlc.New(db),
		db:      db,
	}
}

func (q *Queries) Transaction(ctx context.Context, callback func(ctx context.Context, queries *sqlc.Queries) error) error {
	tx, err := q.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err = callback(ctx, q.WithTx(tx)); err != nil {
		return err
	}

	return tx.Commit()
}
