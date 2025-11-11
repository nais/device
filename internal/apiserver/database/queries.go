package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/nais/device/internal/apiserver/sqlc"
	"github.com/sirupsen/logrus"
)

type Queries struct {
	*sqlc.Queries
	db  *sql.DB
	log logrus.FieldLogger
}

type Querier interface {
	sqlc.Querier
	Transaction(ctx context.Context, callback func(ctx context.Context, queries *sqlc.Queries) error) error
}

func NewQuerier(db *sql.DB, log logrus.FieldLogger) *Queries {
	return &Queries{
		Queries: sqlc.New(db),
		db:      db,
		log:     log,
	}
}

func (q *Queries) Transaction(ctx context.Context, callback func(ctx context.Context, queries *sqlc.Queries) error) error {
	tx, err := q.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(); err != nil && errors.Is(err, sql.ErrTxDone) {
			q.log.Errorf("transaction rollback error: %v", err)
		}
	}()

	if err = callback(ctx, q.WithTx(tx)); err != nil {
		return err
	}

	return tx.Commit()
}
