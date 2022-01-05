package jita

import (
	"fmt"
	"net/http"

	"github.com/nais/device/pkg/basicauth"
)

type client struct {
	HTTPClient *http.Client
	URL        string
}

type Client interface {
	GetPrivilegedUsersForGateway(gateway string) ([]PrivilegedUser, error)
}

func New(username, password, url string) Client {
	return &client{
		HTTPClient: &http.Client{
			Transport: basicauth.Transport{Password: password, Username: username},
		},
		URL: fmt.Sprintf("%s/%s", url, "api/v1"),
	}
}
