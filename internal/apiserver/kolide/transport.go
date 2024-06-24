package kolide

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/nais/device/internal/apiserver/metrics"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var ErrMaxRetriesExceeded = errors.New("max retries exceeded")

type Transport struct {
	Token             string
	Transport         http.RoundTripper
	DefaultRetryAfter time.Duration
	MaxHttpRetries    int
}

var _ http.RoundTripper = &Transport{}

func NewTransport(token string) *Transport {
	return &Transport{
		Token:             token,
		Transport:         http.DefaultTransport,
		MaxHttpRetries:    10,
		DefaultRetryAfter: time.Second,
	}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.Token)
	req.Header.Set("Content-Type", "application/json")

	for attempt := range t.MaxHttpRetries {
		resp, err := t.Transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		metrics.IncKolideStatusCode(resp.StatusCode)

		if resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return resp, nil
		}
		retryAfter := t.getRetryAfter(resp.Header)
		log.WithFields(logrus.Fields{
			"attempt":      attempt + 1,
			"max_attempts": t.MaxHttpRetries,
			"response":     resp.Status,
			"retry_after":  retryAfter,
		}).Debug("rate limited, sleeping")

		select {
		case <-time.After(retryAfter):
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	}

	return nil, ErrMaxRetriesExceeded
}

func (t *Transport) getRetryAfter(header http.Header) time.Duration {
	limit := header.Get("Ratelimit-Limit")
	remaining := header.Get("Ratelimit-Remaining")
	reset := header.Get("Ratelimit-Reset")
	retryAfter := header.Get("Retry-After")

	if retryAfter == "" {
		return 0
	}

	log.WithFields(logrus.Fields{
		"limit":      limit,
		"remaining":  remaining,
		"reset":      reset,
		"retryAfter": retryAfter,
	}).Debug("rate-limited")

	seconds, err := strconv.Atoi(retryAfter)
	if err != nil {
		retryAfterDate, err := time.Parse(time.RFC1123, retryAfter)
		if err != nil || retryAfterDate.Before(time.Now()) {
			return t.DefaultRetryAfter
		}

		return time.Second + time.Until(retryAfterDate).Round(time.Second)
	}

	if seconds < 0 {
		return t.DefaultRetryAfter
	}

	return time.Second * time.Duration(seconds)
}
