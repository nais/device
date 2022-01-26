package auth

type mockApikeyAuthenticator struct {
}

func NewMockAPIKeyAuthenticator() UsernamePasswordAuthenticator {
	return &mockApikeyAuthenticator{}
}

func (a *mockApikeyAuthenticator) Authenticate(username, password string) error {
	return nil
}
