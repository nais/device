package enroller

import (
	"github.com/nais/device/apiserver/database"
	"net/http"
)

type Enroller struct {
	Client             *http.Client
	DB                 *database.APIServerDB
	BootstrapAPIURL    string
	APIServerPublicKey string
	APIServerEndpoint  string
}
