package kolide_test

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	kolideclient "github.com/nais/kolide-event-handler/pkg/kolide"
	"github.com/stretchr/testify/assert"
)

func TestURL(t *testing.T) {
	u, err := url.Parse("https://k2.kolide.com/api/v0/devices/25215")
	assert.NoError(t, err)

	values := u.Query()
	values.Set("foo", "bar")
	u.RawQuery = values.Encode()
	assert.Equal(t, "https://k2.kolide.com/api/v0/devices/25215?foo=bar", u.String())
}

func TestClient(t *testing.T) {
	kolideClient := kolideclient.New(os.Getenv("KOLIDE_API_TOKEN"))
	ctx := context.Background()

	t.Run("get device", func(t *testing.T) {
		t.Skip()
		_, err := kolideClient.GetDevice(ctx, 25215)
		assert.NoError(t, err)
	})

	t.Run("get check", func(t *testing.T) {
		t.Skip()
		_, err := kolideClient.GetCheck(ctx, 27680)
		assert.NoError(t, err)
	})

	t.Run("get failure", func(t *testing.T) {
		t.Skip()
		deviceFailure, err := kolideClient.GetDeviceFailure(ctx, 27066, 123)
		t.Logf("device: %+v", deviceFailure)
		assert.Error(t, err)
	})

	t.Run("rate limit test", func(t *testing.T) {
		tests := []struct {
			name        string
			header      http.Header
			retryAfter  time.Duration
			compareFunc func(got, want time.Duration) bool
		}{
			{
				name:       "no headers should give 0",
				header:     http.Header{},
				retryAfter: 0,
			},
			{
				name: "correct header should give value",
				header: http.Header{
					"Retry-After": []string{"5"},
				},
				retryAfter: 5 * time.Second,
			},
			{
				name: "invalid header should give default value",
				header: http.Header{
					"Retry-After": []string{"a"},
				},
				retryAfter: kolideclient.DefaultRetryAfter,
			},
			{
				name: "negative header should give default value",
				header: http.Header{
					"Retry-After": []string{"-4"},
				},
				retryAfter: kolideclient.DefaultRetryAfter,
			},
			{
				name: "retry-after in the past should give default",
				header: http.Header{
					"Retry-After": []string{time.Now().Add(-time.Hour).Format(time.RFC1123)},
				},
				retryAfter: kolideclient.DefaultRetryAfter,
			},
			{
				name: "retry-after in the future should give delta",
				header: http.Header{
					"Retry-After": []string{time.Now().Add(time.Hour).Format(time.RFC1123)},
				},
				retryAfter: time.Hour,
				compareFunc: func(got, want time.Duration) bool {
					// if calling GetRetryAfter takes > 1 second test will fail, however.. that should not happen.
					return want-got <= time.Second
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := kolideclient.GetRetryAfter(tt.header)
				if tt.compareFunc != nil {
					if !tt.compareFunc(got, tt.retryAfter) {
						t.Errorf("GetRetryAfter() = %v, want %v (using tt.compareFunc)", got, tt.retryAfter)
					}
				} else {
					if got != tt.retryAfter {
						t.Errorf("GetRetryAfter() = %v, want %v", got, tt.retryAfter)
					}
				}
			})
		}
	})

	t.Run("get devices", func(t *testing.T) {
		t.Skip()
		log.SetLevel(log.DebugLevel)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		devices, err := kolideClient.GetDevices(ctx)
		assert.NoError(t, err)
		t.Logf("devices: %+v", len(devices))
		t.Logf("device sample: %+v", devices[len(devices)-1])
	})
}
