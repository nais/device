package integrationtest_test

import (
	"testing"

	"github.com/nais/device/pkg/apiserver/database"
	"github.com/nais/device/pkg/apiserver/testdatabase"
)

func NewDB(t *testing.T) database.APIServer {
	return testdatabase.Setup(t)
}
