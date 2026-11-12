package runtimeconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnsureEnrolled_WithoutTokens(t *testing.T) {
	rc := &runtimeConfig{}

	err := rc.EnsureEnrolled(context.Background(), "serial")

	assert.ErrorContains(t, err, "not authenticated")
}
