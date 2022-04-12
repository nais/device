package enroller

import (
	"net/http"

	"github.com/nais/device/pkg/apiserver/database"
)

type Enroller struct {
	Client             *http.Client
	DB                 database.APIServer
	BootstrapAPIURL    string
	APIServerPublicKey string
	APIServerEndpoint  string
	APIServerIP        string
}
