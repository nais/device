package jita

import (
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
		Url: url,
	}
}
