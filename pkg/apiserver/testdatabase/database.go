package testdatabase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/random"
)

// New creates and returns a new nais device database within the provided database instance
func New(ctx context.Context, ipAllocator database.IPAllocator) (database.APIServer, error) {
	dsn, err := createDatabase()
	if err != nil {
		return nil, fmt.Errorf("creating database: %w", err)
	}

	return database.New(ctx, dsn, ipAllocator, false)
}

func createDatabase() (string, error) {
	dsn := fmt.Sprintf("postgres://postgres:postgres@localhost:5433?sslmode=disable")
	initialConn, err := connect(dsn)
	if err != nil {
		log.Fatal("unable to connect to database")
	}

	defer initialConn.Close()

	databaseName := random.RandomString(5, random.LowerCaseLetters)
	if _, err = initialConn.Exec(fmt.Sprintf("CREATE DATABASE %s", databaseName)); err != nil {
		return "", fmt.Errorf("creating database: %w", err)
	}

	return dsn + fmt.Sprintf("&dbname=%s", databaseName), nil
}

func connect(dsn string) (*sql.DB, error) {
	var initialConn *sql.DB
	var err error

	for i := 0; i < 5; i++ {
		initialConn, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Errorf("[attempt %d/5]: connecting to database: %v", i, err)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("opening connection to database: %w", err)
	}

	for i := 0; i < 5; i++ {
		if err = initialConn.Ping(); err == nil {
			return initialConn, nil
		} else {
			log.Errorf("[attempt %d/5]: pinging database: %v", i, err)
			time.Sleep(1 * time.Second)
		}
	}

	return nil, fmt.Errorf("pinging database: %w", err)
}
