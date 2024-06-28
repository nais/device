package auth

import "context"

type mockApikeyAuthenticator struct{}

func NewMockAPIKeyAuthenticator() UsernamePasswordAuthenticator {
	return &mockApikeyAuthenticator{}
}

func (a *mockApikeyAuthenticator) Authenticate(_ context.Context, username, password string) error {
	return nil
}
