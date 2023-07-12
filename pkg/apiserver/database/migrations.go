package database

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/nais/device/pkg/apiserver/database/schema"
	log "github.com/sirupsen/logrus"
)

func runMigrations(dbPath string) error {
	sourceDriver, err := iofs.New(schema.FS, ".")
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceDriver.Close(); err != nil {
			log.WithError(err).Errorf("close database connection")
		}
	}()

	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, "sqlite3://"+dbPath)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
