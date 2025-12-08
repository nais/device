package database

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/nais/device/internal/apiserver/database/schema"
	"github.com/nais/device/internal/ioconvenience"
	"github.com/sirupsen/logrus"
)

func runMigrations(dbPath string, log logrus.FieldLogger) error {
	sourceDriver, err := iofs.New(schema.FS, ".")
	if err != nil {
		return err
	}
	defer ioconvenience.CloseWithLog(sourceDriver, log)

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
