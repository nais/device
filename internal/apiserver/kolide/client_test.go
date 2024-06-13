package kolide_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nais/device/internal/apiserver/kolide"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	ctx := context.Background()

	// Create a new client

	t.Run("smoke screen client test", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/devices":
				fmt.Fprintf(w, `{}`)
			case "/checks":
				fmt.Fprintf(w, `{}`)
			case "/failures/open":
				fmt.Fprintf(w, `{}`)
			default:
				t.Errorf("unexpected request to %v", r.URL.Path)
			}
		}))

		client := kolide.New("token", logrus.New(), kolide.WithBaseUrl(s.URL))
		err := client.RefreshCache(ctx)
		assert.NoError(t, err)
	})
}
