package auth

type mockApikeyAuthenticator struct {
}

func NewMockAPIKeyAuthenticator() APIKeyAuthenticator {
	return &mockApikeyAuthenticator{}
}

func (a *mockApikeyAuthenticator) Authenticate(username, password string) error {
	return nil
}
