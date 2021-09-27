package auth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/nais/device/pkg/apiserver/config"

	"github.com/stretchr/testify/assert"

	"github.com/nais/device/pkg/apiserver/auth"
)

func TestSessions_AuthURL(t *testing.T) {
	authenticator := auth.NewAuthenticator(
		config.Config{
			Azure: config.Azure{
				Tenant: "62366534-1ec3-4962-8869-9b5535279d0b",
			},
		},
		nil,
		nil,
		nil,
	)

	customPort := 51821
	defaultPort := 51800

	t.Run("custom port authUrl", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "http://whocares", nil)
		request.Header.Set("x-naisdevice-listen-port", strconv.Itoa(customPort))

		response := httptest.NewRecorder()
		authenticator.AuthURL(response, request)

		authUrlFormat := "https://login.microsoftonline.com/62366534-1ec3-4962-8869-9b5535279d0b/oauth2/v2.0/authorize?access_type=offline&client_id=&redirect_uri=http%%3A%%2F%%2Flocalhost%%3A%d&response_type=code&scope=openid+%%2F.default"

		authUrl := response.Body.String()
		authUrlWithoutState := strings.Split(authUrl, "&state=")[0]
		assert.Equal(t, fmt.Sprintf(authUrlFormat, customPort), authUrlWithoutState)
	})

	t.Run("default port authUrl", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "http://whocares", nil)

		response := httptest.NewRecorder()
		authenticator.AuthURL(response, request)

		authUrlFormat := "https://login.microsoftonline.com/62366534-1ec3-4962-8869-9b5535279d0b/oauth2/v2.0/authorize?access_type=offline&client_id=&redirect_uri=http%%3A%%2F%%2Flocalhost%%3A%d&response_type=code&scope=openid+%%2F.default"

		authUrl := response.Body.String()
		authUrlWithoutState := strings.Split(authUrl, "&state=")[0]
		assert.Equal(t, fmt.Sprintf(authUrlFormat, defaultPort), authUrlWithoutState)
	})
}
