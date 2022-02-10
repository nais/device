package auth

import (
	"errors"
)

var ErrInvalidAuth = errors.New("invalid username or password")

type UsernamePasswordAuthenticator interface {
	Authenticate(username, password string) error
}

type apikeyAuthenticator struct {
	users map[string]string
}

func NewAPIKeyAuthenticator(users map[string]string) UsernamePasswordAuthenticator {
	return &apikeyAuthenticator{
		users: users,
	}
}

func (a *apikeyAuthenticator) Authenticate(username, password string) error {
	if len(username) > 0 && len(password) > 0 && a.users[username] == password {
		return nil
	}
	return ErrInvalidAuth
}
