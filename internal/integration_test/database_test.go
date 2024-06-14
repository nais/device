package integrationtest_test

import (
	"testing"

	"github.com/nais/device/internal/apiserver/database"
	"github.com/nais/device/internal/apiserver/testdatabase"
)

func NewDB(t *testing.T) database.Database {
	return testdatabase.Setup(t)
}
