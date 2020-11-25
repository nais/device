package jita

import (
	"fmt"
	"github.com/nais/device/pkg/basicauth"
	"net/http"
)

type Jita struct {
	HTTPClient *http.Client
	Url        string
}

func New(username, password, url string) *Jita {
	return &Jita{
		HTTPClient: &http.Client{
			Transport: basicauth.Transport{Password: password, Username: username},
		},
		Url: fmt.Sprintf("%s/%s", url, "api/v1"),
	}
}
