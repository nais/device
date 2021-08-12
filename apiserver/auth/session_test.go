package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/nais/device/apiserver/auth"
	"github.com/nais/device/apiserver/database"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

func TestSessions_AuthURL(t *testing.T) {
	sessions := auth.Sessions{
		DB:     nil,
		Active: make(map[string]*database.SessionInfo),
		State:  make(map[string]bool),
		OAuthConfig: &oauth2.Config{
			RedirectURL:  "http://localhost",
			ClientID:     "{client_id}",
			ClientSecret: "{client_secret}",
			Scopes:       []string{"openid", fmt.Sprintf("%s/.default", "{client_id}")},
			Endpoint:     endpoints.AzureAD("{tenant_id}"),
		},
	}

	customPort := 51821
	defaultPort := 51800

	t.Run("custom port authUrl", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "http://whocares", nil)
		request.Header.Set("x-naisdevice-listen-port", strconv.Itoa(customPort))

		response := httptest.NewRecorder()
		sessions.AuthURL(response, request)

		authUrlFormat := "https://login.microsoftonline.com/%%7Btenant_id%%7D/oauth2/v2.0/authorize?access_type=offline&client_id=%%7Bclient_id%%7D&redirect_uri=http%%3A%%2F%%2Flocalhost%%3A%d&response_type=code&scope=openid+%%7Bclient_id%%7D%%2F.default"

		authUrl := response.Body.String()
		authUrlWithoutState := strings.Split(authUrl, "&state=")[0]
		assert.Equal(t, fmt.Sprintf(authUrlFormat, customPort), authUrlWithoutState)
	})

	t.Run("default port authUrl", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "http://whocares", nil)

		response := httptest.NewRecorder()
		sessions.AuthURL(response, request)

		authUrlFormat := "https://login.microsoftonline.com/%%7Btenant_id%%7D/oauth2/v2.0/authorize?access_type=offline&client_id=%%7Bclient_id%%7D&redirect_uri=http%%3A%%2F%%2Flocalhost%%3A%d&response_type=code&scope=openid+%%7Bclient_id%%7D%%2F.default"

		authUrl := response.Body.String()
		authUrlWithoutState := strings.Split(authUrl, "&state=")[0]
		assert.Equal(t, fmt.Sprintf(authUrlFormat, defaultPort), authUrlWithoutState)
	})
}
