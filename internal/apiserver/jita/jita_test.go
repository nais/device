package jita_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nais/device/internal/apiserver/jita"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestJita(t *testing.T) {
	log := logrus.StandardLogger().WithField("component", "test")
	t.Run("response with data", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/gatewaysAccess", func(rw http.ResponseWriter, _ *http.Request) {
			response := `{
			"onprem-k8s-prod": [
			{
				"user_id": "8e81112f-638e-4efb-a02f-9662a17ab38b",
				"expires": "2022-02-14 17:12:29+00",
				"ttl": 5631
			},
			{
				"user_id": "516e0fc5-eae4-4918-b692-750618bbd3ab",
				"expires": "2022-02-14 16:08:53+00",
				"ttl": 1815
			}
			],
			"naisvakt": [
			{
				"user_id": "fbecc223-3967-4c1c-ae0e-ca74c11fa340",
				"expires": "2022-02-14 20:18:20+00",
				"ttl": 16782
			}
			]
		}`

			_, err := rw.Write([]byte(response))
			assert.NoError(t, err)
		})

		s := httptest.NewServer(mux)
		defer s.Close()

		j := jita.New(log, "", "", s.URL)
		err := j.UpdatePrivilegedUsers()
		assert.NoError(t, err)

		usersOnprem := j.GetPrivilegedUsersForGateway("onprem-k8s-prod")
		expectedOnprem := []jita.PrivilegedUser{
			{UserId: "8e81112f-638e-4efb-a02f-9662a17ab38b"},
			{UserId: "516e0fc5-eae4-4918-b692-750618bbd3ab"},
		}
		assert.Equal(t, expectedOnprem, usersOnprem)

		usersNaisvakt := j.GetPrivilegedUsersForGateway("naisvakt")
		expectedNaisvakt := []jita.PrivilegedUser{
			{UserId: "fbecc223-3967-4c1c-ae0e-ca74c11fa340"},
		}

		assert.Equal(t, expectedNaisvakt, usersNaisvakt)

		usersEmpty := j.GetPrivilegedUsersForGateway("empty")
		var expectedEmpty []jita.PrivilegedUser

		assert.Equal(t, expectedEmpty, usersEmpty)
	})

	t.Run("empty response", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/gatewaysAccess", func(rw http.ResponseWriter, _ *http.Request) {
			response := `{}`

			_, err := rw.Write([]byte(response))
			assert.NoError(t, err)
		})

		s := httptest.NewServer(mux)
		defer s.Close()

		j := jita.New(log, "", "", s.URL)
		err := j.UpdatePrivilegedUsers()
		assert.NoError(t, err)

		var expectedEmpty []jita.PrivilegedUser
		usersEmpty := j.GetPrivilegedUsersForGateway("empty")

		assert.Equal(t, expectedEmpty, usersEmpty)
	})
}
