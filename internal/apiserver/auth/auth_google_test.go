package auth_test

import (
	"net/http"
	"testing"

	"github.com/nais/device/internal/apiserver/auth"
	"github.com/nais/device/internal/token"
	"github.com/nais/device/pkg/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeParser map[string]*token.User

func (p fakeParser) ParseString(s string) (*token.User, error) {
	user, ok := p[s]
	if !ok {
		return nil, assert.AnError
	}
	return user, nil
}

func (p fakeParser) ParseHeader(http.Header, string) (*token.User, error) {
	return nil, assert.AnError
}

func TestGoogleAuth_ValidateJita(t *testing.T) {
	parser := fakeParser{
		"valid-token":      {ID: "user-1", Email: "user@example.com"},
		"other-user-token": {ID: "user-2", Email: "other@example.com"},
		"empty-id-token":   {ID: "", Email: "user@example.com"},
	}

	authenticator := auth.NewGoogleAuthenticator(parser, nil, nil)
	session := &pb.Session{ObjectID: "user-1"}

	tests := []struct {
		name    string
		session *pb.Session
		token   string
		wantErr string
	}{
		{name: "token for session user", session: session, token: "valid-token"},
		{name: "token for another user", session: session, token: "other-user-token", wantErr: "does not match"},
		{name: "unparsable token", session: session, token: "invalid-signature", wantErr: "parse JITA token"},
		{name: "empty token", session: session, token: "", wantErr: "parse JITA token"},
		{name: "empty ids never match", session: &pb.Session{ObjectID: ""}, token: "empty-id-token", wantErr: "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := authenticator.ValidateJita(tt.session, tt.token)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
