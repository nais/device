package database

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/nais/device/pkg/apiserver/sqlc"
	log "github.com/sirupsen/logrus"
)

type Queries struct {
	*sqlc.Queries
	conn *pgx.Conn
}

type Querier interface {
	sqlc.Querier
	Transaction(ctx context.Context, callback func(ctx context.Context, queries *sqlc.Queries) error) error
}

func NewQuerier(conn *pgx.Conn) *Queries {
	return &Queries{
		Queries: sqlc.New(conn),
		conn:    conn,
	}
}

func (q *Queries) Transaction(ctx context.Context, callback func(ctx context.Context, queries *sqlc.Queries) error) error {
	tx, err := q.conn.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			log.WithError(err).Errorf("rollback")
		}
	}()

	if err = callback(ctx, q.WithTx(tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
