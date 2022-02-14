package jita

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/nais/device/pkg/basicauth"
)

type client struct {
	HTTPClient      *http.Client
	URL             string
	PrivilegedUsers map[string][]PrivilegedUser
	lock            sync.RWMutex
}

type Client interface {
	GetPrivilegedUsersForGateway(gateway string) []PrivilegedUser
	UpdatePrivilegedUsers() error
}

func New(username, password, url string) Client {
	return &client{
		HTTPClient: &http.Client{
			Transport: basicauth.Transport{Password: password, Username: username},
		},
		URL:             fmt.Sprintf("%s/%s", url, "api/v1"),
		PrivilegedUsers: make(map[string][]PrivilegedUser, 0),
	}
}
