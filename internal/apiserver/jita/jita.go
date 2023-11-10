package jita

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/nais/device/internal/basicauth"
	"github.com/sirupsen/logrus"
)

type client struct {
	HTTPClient      *http.Client
	URL             string
	PrivilegedUsers map[string][]PrivilegedUser
	lock            sync.RWMutex
	log             *logrus.Entry
}

type Client interface {
	GetPrivilegedUsersForGateway(gateway string) []PrivilegedUser
	UpdatePrivilegedUsers() error
}

func New(log *logrus.Entry, username, password, url string) Client {
	return &client{
		HTTPClient: &http.Client{
			Transport: basicauth.Transport{Password: password, Username: username},
		},
		URL:             fmt.Sprintf("%s/%s", url, "api/v1"),
		PrivilegedUsers: make(map[string][]PrivilegedUser, 0),
		log:             log,
	}
}
